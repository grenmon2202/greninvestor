package charts

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/grenmon2202/greninvestor/config"
	"github.com/grenmon2202/greninvestor/logging"
	"go.uber.org/zap"
)

type YahooClient struct {
	http *http.Client
}

func NewYahooClient() *YahooClient {
	return &YahooClient{
		http: &http.Client{Timeout: 12 * time.Second},
	}
}

func (c *YahooClient) FetchCandles(ctx context.Context, symbol, rangeStr, interval string) ([]Candle, error) {
	logging.Init()
	defer logging.L.Sync()

	logging.L.Info("Fetching candles", zap.String("symbol", symbol), zap.String("range", rangeStr), zap.String("interval", interval))

	u, _ := url.Parse(config.GetYahooURL(symbol))

	q := u.Query()
	q.Set("range", rangeStr)
	q.Set("interval", interval)
	u.RawQuery = q.Encode()

	req, _ := http.NewRequestWithContext(ctx, "GET", u.String(), nil)

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; greninvestor/1.0)")

	resp, err := c.http.Do(req)

	if err != nil {
		logging.L.Error("HTTP request failed", zap.String("symbol", symbol), zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		logging.L.Error("Non-200 HTTP response", zap.String("symbol", symbol), zap.Int("status", resp.StatusCode), zap.String("body", string(b)))
		return nil, fmt.Errorf("yahoo http %d: %s", resp.StatusCode, string(b))
	}

	var cr chartResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		logging.L.Error("Failed to decode JSON response", zap.String("symbol", symbol), zap.Error(err))
		return nil, err
	}

	if cr.Chart.Error != nil {
		logging.L.Error("Yahoo API returned error", zap.String("symbol", symbol), zap.Any("error", cr.Chart.Error))
		return nil, fmt.Errorf("yahoo api error: %v", cr.Chart.Error)
	}

	if len(cr.Chart.Result) == 0 {
		logging.L.Error("Yahoo API returned no results", zap.String("symbol", symbol))
		return nil, fmt.Errorf("yahoo api returned no results")
	}

	r := cr.Chart.Result[0]

	if len(r.Indicators.Quote) == 0 {
		logging.L.Error("Yahoo API returned no quote data", zap.String("symbol", symbol))
		return nil, fmt.Errorf("yahoo api returned no quote data")
	}

	qt := r.Indicators.Quote[0]

	out := make([]Candle, 0, len(r.Timestamp))
	for i, ts := range r.Timestamp {
		if i >= len(qt.Open) || i >= len(qt.High) || i >= len(qt.Low) || i >= len(qt.Close) || i >= len(qt.Volume) {
			continue
		}
		if qt.Open[i] == nil || qt.High[i] == nil || qt.Low[i] == nil || qt.Close[i] == nil || qt.Volume[i] == nil {
			continue
		}

		out = append(out, Candle{
			T: time.Unix(ts, 0),
			O: *qt.Open[i],
			H: *qt.High[i],
			L: *qt.Low[i],
			C: *qt.Close[i],
			V: *qt.Volume[i],
		})
	}

	if len(out) == 0 {
		logging.L.Error("Yahoo API returned no complete candle data", zap.String("symbol", symbol))
		return nil, fmt.Errorf("yahoo api returned no complete candle data")
	}

	return out, nil
}
