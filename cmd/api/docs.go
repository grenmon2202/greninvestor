package main

import (
	"encoding/json"
	"html/template"
	"net/http"
)

type apiDocEndpoint struct {
	Method      string   `json:"method"`
	Path        string   `json:"path"`
	Description string   `json:"description"`
	QueryParams []string `json:"query_params,omitempty"`
}

type apiDocs struct {
	Name      string           `json:"name"`
	Version   string           `json:"version"`
	BasePath  string           `json:"base_path"`
	Endpoints []apiDocEndpoint `json:"endpoints"`
}

var dashboardAPIDocs = apiDocs{
	Name:     "Greninvestor Dashboard API",
	Version:  "v1",
	BasePath: "/",
	Endpoints: []apiDocEndpoint{
		{Method: "GET", Path: "/health", Description: "Service liveness check with current server time."},
		{Method: "GET", Path: "/docs", Description: "Human-readable API documentation."},
		{Method: "GET", Path: "/openapi.json", Description: "Machine-readable API documentation JSON."},
		{Method: "GET", Path: "/api/summary", Description: "Top-level dashboard summary including latest run and enabled portfolio cards."},
		{Method: "GET", Path: "/api/portfolios", Description: "Latest portfolio cards for all enabled portfolios."},
		{Method: "GET", Path: "/api/portfolio/:name/overview", Description: "Latest snapshot for one enabled portfolio."},
		{Method: "GET", Path: "/api/portfolio/:name/history", Description: "Portfolio history time series for one enabled portfolio.", QueryParams: []string{"limit", "from_ts", "to_ts"}},
		{Method: "GET", Path: "/api/portfolio/:name/holdings", Description: "Current enriched holdings for one enabled portfolio."},
		{Method: "GET", Path: "/api/portfolio/:name/trades", Description: "Trade history for one enabled portfolio.", QueryParams: []string{"limit", "from_ts", "to_ts"}},
		{Method: "GET", Path: "/api/runs/latest", Description: "Latest executor run summary."},
		{Method: "GET", Path: "/api/runs/history", Description: "Executor run history.", QueryParams: []string{"limit"}},
	},
}

var docsTemplate = template.Must(template.New("docs").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{.Name}}</title>
  <style>
    body { font-family: Arial, sans-serif; max-width: 960px; margin: 40px auto; padding: 0 16px; color: #1f2937; }
    h1 { margin-bottom: 8px; }
    p { color: #4b5563; }
    table { width: 100%; border-collapse: collapse; margin-top: 24px; }
    th, td { text-align: left; padding: 12px; border-bottom: 1px solid #e5e7eb; vertical-align: top; }
    th { background: #f9fafb; }
    code { background: #f3f4f6; padding: 2px 6px; border-radius: 4px; }
  </style>
</head>
<body>
  <h1>{{.Name}}</h1>
  <p>Version {{.Version}}. JSON docs are also available at <code>/openapi.json</code>.</p>
  <table>
    <thead>
      <tr>
        <th>Method</th>
        <th>Path</th>
        <th>Description</th>
        <th>Query Params</th>
      </tr>
    </thead>
    <tbody>
      {{range .Endpoints}}
      <tr>
        <td><code>{{.Method}}</code></td>
        <td><code>{{.Path}}</code></td>
        <td>{{.Description}}</td>
        <td>{{if .QueryParams}}{{range $index, $param := .QueryParams}}{{if $index}}, {{end}}<code>{{$param}}</code>{{end}}{{else}}-{{end}}</td>
      </tr>
      {{end}}
    </tbody>
  </table>
</body>
</html>`))

func handleDocs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = docsTemplate.Execute(w, dashboardAPIDocs)
}

func handleOpenAPIDocs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(dashboardAPIDocs)
}
