package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/grenmon2202/greninvestor/config"
	"github.com/grenmon2202/greninvestor/db"
	"github.com/grenmon2202/greninvestor/logging"
	"go.uber.org/zap"
)

type apiServer struct {
	enabledPortfolioSet map[string]struct{}
	nowFn               func() time.Time
}

func newAPIServer() (*apiServer, error) {
	enabled := config.GetEnabledStrategies()
	portfolioSet := make(map[string]struct{}, len(enabled))
	for _, strategy := range enabled {
		portfolioSet[strategy.PortfolioName] = struct{}{}
	}

	return &apiServer{
		enabledPortfolioSet: portfolioSet,
		nowFn:               time.Now,
	}, nil
}

func main() {
	logging.Init()
	defer logging.L.Sync()

	server, err := newAPIServer()
	if err != nil {
		logging.L.Fatal("Failed to initialize API server", zap.Error(err))
	}

	addr := os.Getenv("API_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleDocs)
	mux.HandleFunc("/docs", handleDocs)
	mux.HandleFunc("/openapi.json", handleOpenAPIDocs)
	mux.HandleFunc("/health", server.handleHealth)
	mux.HandleFunc("/api/summary", server.handleSummary)
	mux.HandleFunc("/api/portfolios", server.handlePortfolios)
	mux.HandleFunc("/api/portfolio/", server.handlePortfolioRoutes)
	mux.HandleFunc("/api/runs/latest", server.handleRunsLatest)
	mux.HandleFunc("/api/runs/history", server.handleRunsHistory)

	logging.L.Info("Starting API server", zap.String("addr", addr))
	if err := http.ListenAndServe(addr, mux); err != nil {
		logging.L.Fatal("API server exited", zap.Error(err))
	}
}

func (s *apiServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":      "ok",
		"server_time": s.nowFn().Unix(),
	})
}

func (s *apiServer) handleSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	latestRun, err := db.FetchLatestRun()
	latestRunFound := true
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if errors.Is(err, sql.ErrNoRows) {
		latestRunFound = false
	}

	portfolios, err := s.fetchPortfolioCards()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	totalOpenHoldings := 0
	for _, name := range s.enabledPortfolioNames() {
		holdings, err := db.FetchEnrichedHoldings(name, s.nowFn())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		totalOpenHoldings += len(holdings)
	}

	resp := map[string]any{
		"latest_run":          nil,
		"portfolios":          portfolios,
		"enabled_portfolios":  len(portfolios),
		"total_open_holdings": totalOpenHoldings,
	}
	if latestRunFound {
		resp["latest_run"] = latestRun
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *apiServer) handlePortfolios(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	portfolios, err := s.fetchPortfolioCards()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"portfolios": portfolios})
}

func (s *apiServer) handlePortfolioRoutes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/portfolio/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	name, resource := parts[0], parts[1]
	if !s.isEnabledPortfolio(name) {
		writeError(w, http.StatusNotFound, "portfolio not found")
		return
	}

	switch resource {
	case "overview":
		s.handlePortfolioOverview(w, r, name)
	case "history":
		s.handlePortfolioHistory(w, r, name)
	case "holdings":
		s.handlePortfolioHoldings(w, r, name)
	case "trades":
		s.handlePortfolioTrades(w, r, name)
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (s *apiServer) handlePortfolioOverview(w http.ResponseWriter, r *http.Request, name string) {
	record, err := db.FetchLatestPortfolioHistory(name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "portfolio snapshot not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, record)
}

func (s *apiServer) handlePortfolioHistory(w http.ResponseWriter, r *http.Request, name string) {
	limit, fromTs, toTs, ok := parseHistoryQuery(w, r)
	if !ok {
		return
	}

	records, err := db.FetchPortfolioHistory(name, limit, fromTs, toTs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"portfolio": name, "history": records})
}

func (s *apiServer) handlePortfolioHoldings(w http.ResponseWriter, r *http.Request, name string) {
	holdings, err := db.FetchEnrichedHoldings(name, s.nowFn())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"portfolio": name, "holdings": holdings})
}

func (s *apiServer) handlePortfolioTrades(w http.ResponseWriter, r *http.Request, name string) {
	limit, fromTs, toTs, ok := parseHistoryQuery(w, r)
	if !ok {
		return
	}

	trades, err := db.FetchTrades(name, limit, fromTs, toTs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"portfolio": name, "trades": trades})
}

func (s *apiServer) handleRunsLatest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	record, err := db.FetchLatestRun()
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "run not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, record)
}

func (s *apiServer) handleRunsHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	limit, err := parseIntQuery(r, "limit", 100)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	records, err := db.FetchRunHistory(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"runs": records})
}

func (s *apiServer) fetchPortfolioCards() ([]db.PortfolioHistoryRecord, error) {
	names := s.enabledPortfolioNames()
	out := make([]db.PortfolioHistoryRecord, 0, len(names))
	for _, name := range names {
		record, err := db.FetchLatestPortfolioHistory(name)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return nil, err
		}
		out = append(out, record)
	}
	return out, nil
}

func (s *apiServer) enabledPortfolioNames() []string {
	names := make([]string, 0, len(s.enabledPortfolioSet))
	for name := range s.enabledPortfolioSet {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func (s *apiServer) isEnabledPortfolio(name string) bool {
	_, ok := s.enabledPortfolioSet[name]
	return ok
}

func parseHistoryQuery(w http.ResponseWriter, r *http.Request) (int, int64, int64, bool) {
	limit, err := parseIntQuery(r, "limit", 200)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return 0, 0, 0, false
	}

	fromTs, err := parseInt64Query(r, "from_ts", 0)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return 0, 0, 0, false
	}

	toTs, err := parseInt64Query(r, "to_ts", 0)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return 0, 0, 0, false
	}

	return limit, fromTs, toTs, true
}

func parseIntQuery(r *http.Request, key string, fallback int) (int, error) {
	value := r.URL.Query().Get(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, errors.New("invalid " + key)
	}
	return parsed, nil
}

func parseInt64Query(r *http.Request, key string, fallback int64) (int64, error) {
	value := r.URL.Query().Get(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed < 0 {
		return 0, errors.New("invalid " + key)
	}
	return parsed, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": message})
}
