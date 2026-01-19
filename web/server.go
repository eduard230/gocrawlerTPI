package web

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"gocrawler/storage"
)

// Server represents the web dashboard server
type Server struct {
	port     int
	results  *storage.Results
	template *template.Template
}

// NewServer creates a new Server instance
func NewServer(port int, results *storage.Results) *Server {
	// Parse template once at startup for security and performance
	tmpl := template.Must(template.New("dashboard").Parse(dashboardHTML))

	return &Server{
		port:     port,
		results:  results,
		template: tmpl,
	}
}

// Start starts the web server
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Serve static files
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/stats", s.handleStats)
	mux.HandleFunc("/api/pages", s.handlePages)

	addr := fmt.Sprintf(":%d", s.port)
	fmt.Printf("🌐 Dashboard starting on http://localhost%s\n", addr)

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return server.ListenAndServe()
}

// handleIndex serves the main dashboard HTML
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Use template execution for proper HTML escaping (security best practice)
	if err := s.template.Execute(w, nil); err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
	}
}

// handleStats returns crawling statistics as JSON
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	stats := s.results.GetStats()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// handlePages returns all crawled pages as JSON
func (s *Server) handlePages(w http.ResponseWriter, r *http.Request) {
	pages := s.results.GetPages()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pages)
}

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Go Crawler Dashboard</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: #333;
            padding: 20px;
            min-height: 100vh;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            border-radius: 15px;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            overflow: hidden;
        }
        header {
            background: linear-gradient(135deg, #5a67d8 0%, #6b46c1 100%);
            color: white;
            padding: 30px;
            text-align: center;
        }
        h1 { font-size: 2.5em; margin-bottom: 10px; }
        .subtitle { opacity: 0.9; font-size: 1.1em; }
        .stats {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            padding: 30px;
            background: #f7fafc;
        }
        .stat-card {
            background: white;
            padding: 20px;
            border-radius: 10px;
            box-shadow: 0 4px 6px rgba(0,0,0,0.1);
            text-align: center;
            transition: transform 0.2s;
        }
        .stat-card:hover { transform: translateY(-5px); }
        .stat-value {
            font-size: 2.5em;
            font-weight: bold;
            color: #5a67d8;
            margin: 10px 0;
        }
        .stat-label {
            color: #718096;
            font-size: 0.9em;
            text-transform: uppercase;
            letter-spacing: 1px;
        }
        .pages-section {
            padding: 30px;
        }
        .pages-section h2 {
            color: #2d3748;
            margin-bottom: 20px;
            font-size: 1.8em;
        }
        .page-item {
            background: #f7fafc;
            padding: 15px;
            margin-bottom: 15px;
            border-radius: 8px;
            border-left: 4px solid #5a67d8;
            transition: all 0.2s;
        }
        .page-item:hover {
            background: #edf2f7;
            transform: translateX(5px);
        }
        .page-item.error {
            border-left-color: #f56565;
        }
        .page-url {
            color: #5a67d8;
            font-weight: bold;
            word-break: break-all;
            margin-bottom: 5px;
        }
        .page-title {
            color: #2d3748;
            margin-bottom: 5px;
        }
        .page-meta {
            color: #718096;
            font-size: 0.85em;
        }
        .page-error {
            color: #f56565;
            font-weight: bold;
            margin-top: 5px;
        }
        .loading {
            text-align: center;
            padding: 40px;
            color: #718096;
            font-size: 1.2em;
        }
        .spinner {
            border: 4px solid #f3f4f6;
            border-top: 4px solid #5a67d8;
            border-radius: 50%;
            width: 40px;
            height: 40px;
            animation: spin 1s linear infinite;
            margin: 20px auto;
        }
        @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>🚀 Go Concurrent Web Crawler</h1>
            <p class="subtitle">Real-time Dashboard - Demonstrating Goroutines & Channels</p>
        </header>

        <div class="stats" id="stats">
            <div class="loading">
                <div class="spinner"></div>
                Loading statistics...
            </div>
        </div>

        <div class="pages-section">
            <h2>📄 Crawled Pages</h2>
            <div id="pages">
                <div class="loading">
                    <div class="spinner"></div>
                    Loading pages...
                </div>
            </div>
        </div>
    </div>

    <script>
        // Auto-refresh every 2 seconds
        function fetchStats() {
            fetch('/api/stats')
                .then(res => res.json())
                .then(data => {
                    document.getElementById('stats').innerHTML = ` + "`" + `
                        <div class="stat-card">
                            <div class="stat-label">Total Pages</div>
                            <div class="stat-value">${data.TotalPages || 0}</div>
                        </div>
                        <div class="stat-card">
                            <div class="stat-label">Unique Links</div>
                            <div class="stat-value">${data.UniqueLinks || 0}</div>
                        </div>
                        <div class="stat-card">
                            <div class="stat-label">Success Rate</div>
                            <div class="stat-value">${data.TotalPages ? Math.round(data.SuccessCount / data.TotalPages * 100) : 0}%</div>
                        </div>
                        <div class="stat-card">
                            <div class="stat-label">Avg Response</div>
                            <div class="stat-value">${Math.round(data.AvgResponseTime || 0)}<span style="font-size: 0.5em;">ms</span></div>
                        </div>
                        <div class="stat-card">
                            <div class="stat-label">Successful</div>
                            <div class="stat-value" style="color: #48bb78;">${data.SuccessCount || 0}</div>
                        </div>
                        <div class="stat-card">
                            <div class="stat-label">Failed</div>
                            <div class="stat-value" style="color: #f56565;">${data.FailCount || 0}</div>
                        </div>
                    ` + "`" + `;
                })
                .catch(err => console.error('Error fetching stats:', err));
        }

        function fetchPages() {
            fetch('/api/pages')
                .then(res => res.json())
                .then(data => {
                    if (!data || data.length === 0) {
                        document.getElementById('pages').innerHTML = '<div class="loading">No pages crawled yet...</div>';
                        return;
                    }

                    document.getElementById('pages').innerHTML = data.map(page => ` + "`" + `
                        <div class="page-item ${page.success ? '' : 'error'}">
                            <div class="page-url">${page.url}</div>
                            ${page.title ? ` + "`<div class=\"page-title\">${page.title}</div>`" + ` : ''}
                            <div class="page-meta">
                                ⏱️ ${page.response_time_ms / 1000000}ms |
                                🔗 ${page.links ? page.links.length : 0} links |
                                📅 ${new Date(page.crawled_at).toLocaleTimeString()}
                            </div>
                            ${!page.success ? ` + "`<div class=\"page-error\">❌ Error: ${page.error}</div>`" + ` : ''}
                        </div>
                    ` + "`" + `).join('');
                })
                .catch(err => console.error('Error fetching pages:', err));
        }

        // Initial fetch
        fetchStats();
        fetchPages();

        // Auto-refresh every 2 seconds
        setInterval(() => {
            fetchStats();
            fetchPages();
        }, 2000);
    </script>
</body>
</html>`
