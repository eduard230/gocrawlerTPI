package crawler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"gocrawler/parser"
	"gocrawler/storage"
)

// Crawler represents a concurrent web crawler
type Crawler struct {
	workers     int
	maxDepth    int
	rateLimiter *RateLimiter
	results     *storage.Results
	visited     map[string]bool
	visitedMu   sync.RWMutex
	client      *http.Client
	startTime   time.Time
}

// Job represents a crawl job
type Job struct {
	URL   string
	Depth int
}

// New creates a new Crawler instance
func New(workers, rateLimit, maxDepth int, results *storage.Results) *Crawler {
	return &Crawler{
		workers:     workers,
		maxDepth:    maxDepth,
		rateLimiter: NewRateLimiter(rateLimit),
		results:     results,
		visited:     make(map[string]bool),
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConnsPerHost: workers,
			},
		},
	}
}

// Crawl starts the crawling process
func (c *Crawler) Crawl(ctx context.Context, startURL string) {
	c.startTime = time.Now()

	// Create job queue (buffered channel)
	jobs := make(chan Job, 100)
	jobsDone := make(chan bool)

	// Create worker pool using goroutines
	var wg sync.WaitGroup
	for i := 0; i < c.workers; i++ {
		wg.Add(1)
		go c.worker(ctx, i, jobs, &wg)
	}

	// Send initial job
	jobs <- Job{URL: startURL, Depth: 0}

	// Monitor goroutine to close jobs channel when done
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		prevVisited := 0
		stableCount := 0

		for {
			select {
			case <-ctx.Done():
				jobsDone <- true
				return
			case <-ticker.C:
				c.visitedMu.RLock()
				currentVisited := len(c.visited)
				c.visitedMu.RUnlock()

				// If no new pages were visited, increment stable counter
				if currentVisited == prevVisited {
					stableCount++
				} else {
					stableCount = 0
				}
				prevVisited = currentVisited

				// If stable for 3 consecutive checks (1.5 seconds), we're done
				if stableCount >= 3 {
					jobsDone <- true
					return
				}
			}
		}
	}()

	// Wait for completion signal then close channel
	<-jobsDone
	close(jobs)

	wg.Wait()
	c.results.SetDuration(time.Since(c.startTime))
	log.Println("🏁 All workers finished")
}

// worker processes jobs from the queue
func (c *Crawler) worker(ctx context.Context, id int, jobs chan Job, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-jobs:
			if !ok {
				return
			}

			// Check if already visited
			if c.isVisited(job.URL) {
				continue
			}
			c.markVisited(job.URL)

			// Rate limiting
			c.rateLimiter.Wait(ctx)

			// Fetch and parse
			start := time.Now()
			resp, err := c.client.Get(job.URL)
			duration := time.Since(start)

			if err != nil {
				c.results.AddPage(job.URL, "", "", nil, duration, err)
				log.Printf("❌ [Worker %d] Error fetching %s: %v", id, job.URL, err)
				continue
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				c.results.AddPage(job.URL, "", "", nil, duration, fmt.Errorf("status %d", resp.StatusCode))
				log.Printf("⚠️  [Worker %d] Non-200 status for %s: %d", id, job.URL, resp.StatusCode)
				continue
			}

			// Parse HTML
			pageInfo, err := parser.Parse(resp.Body, job.URL)
			if err != nil {
				c.results.AddPage(job.URL, "", "", nil, duration, err)
				log.Printf("❌ [Worker %d] Error parsing %s: %v", id, job.URL, err)
				continue
			}

			// Store results
			c.results.AddPage(job.URL, pageInfo.Title, pageInfo.Description, pageInfo.Links, duration, nil)
			log.Printf("✅ [Worker %d] Crawled: %s (depth=%d, links=%d, %dms)",
				id, job.URL, job.Depth, len(pageInfo.Links), duration.Milliseconds())

			// Queue child URLs if depth allows
			if job.Depth < c.maxDepth {
				baseURL, _ := url.Parse(job.URL)
				for _, link := range pageInfo.Links {
					childURL := c.resolveURL(baseURL, link)
					if childURL != "" && c.shouldCrawl(childURL, baseURL) {
						select {
						case jobs <- Job{URL: childURL, Depth: job.Depth + 1}:
						case <-ctx.Done():
							return
						default:
							// Queue full, skip this URL
						}
					}
				}
			}
		}
	}
}

// isVisited checks if URL was already visited (thread-safe)
func (c *Crawler) isVisited(url string) bool {
	c.visitedMu.RLock()
	defer c.visitedMu.RUnlock()
	return c.visited[url]
}

// markVisited marks URL as visited (thread-safe)
func (c *Crawler) markVisited(url string) {
	c.visitedMu.Lock()
	defer c.visitedMu.Unlock()
	c.visited[url] = true
}

// resolveURL resolves relative URLs to absolute
func (c *Crawler) resolveURL(base *url.URL, href string) string {
	link, err := url.Parse(href)
	if err != nil {
		return ""
	}
	return base.ResolveReference(link).String()
}

// shouldCrawl determines if URL should be crawled (same domain only)
func (c *Crawler) shouldCrawl(targetURL string, baseURL *url.URL) bool {
	target, err := url.Parse(targetURL)
	if err != nil {
		return false
	}

	// Only crawl same domain
	return target.Host == baseURL.Host && (target.Scheme == "http" || target.Scheme == "https")
}
