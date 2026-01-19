package storage

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// Page represents a crawled page
type Page struct {
	URL          string        `json:"url"`
	Title        string        `json:"title"`
	Description  string        `json:"description"`
	Links        []string      `json:"links"`
	ResponseTime time.Duration `json:"response_time_ms"`
	Success      bool          `json:"success"`
	Error        string        `json:"error,omitempty"`
	CrawledAt    time.Time     `json:"crawled_at"`
}

// Stats represents crawling statistics
type Stats struct {
	TotalPages      int
	UniqueLinks     int
	AvgResponseTime float64
	SuccessCount    int
	FailCount       int
	Duration        time.Duration
}

// Results stores all crawled pages (thread-safe)
type Results struct {
	pages    []*Page
	mu       sync.RWMutex
	duration time.Duration
}

// NewResults creates a new Results instance
func NewResults() *Results {
	return &Results{
		pages: make([]*Page, 0),
	}
}

// AddPage adds a crawled page to results (thread-safe)
func (r *Results) AddPage(url, title, description string, links []string, responseTime time.Duration, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	page := &Page{
		URL:          url,
		Title:        title,
		Description:  description,
		Links:        links,
		ResponseTime: responseTime,
		Success:      err == nil,
		CrawledAt:    time.Now(),
	}

	if err != nil {
		page.Error = err.Error()
	}

	r.pages = append(r.pages, page)
}

// GetPages returns all pages (thread-safe)
func (r *Results) GetPages() []*Page {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return copy to prevent race conditions
	pages := make([]*Page, len(r.pages))
	copy(pages, r.pages)
	return pages
}

// GetStats calculates and returns statistics
func (r *Results) GetStats() Stats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := Stats{
		TotalPages: len(r.pages),
		Duration:   r.duration,
	}

	if stats.TotalPages == 0 {
		return stats
	}

	var totalTime time.Duration
	uniqueLinks := make(map[string]bool)

	for _, page := range r.pages {
		totalTime += page.ResponseTime
		if page.Success {
			stats.SuccessCount++
		} else {
			stats.FailCount++
		}

		for _, link := range page.Links {
			uniqueLinks[link] = true
		}
	}

	stats.UniqueLinks = len(uniqueLinks)
	stats.AvgResponseTime = float64(totalTime.Milliseconds()) / float64(stats.TotalPages)

	return stats
}

// SetDuration sets the total crawl duration
func (r *Results) SetDuration(d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.duration = d
}

// ExportJSON exports results to JSON file
func (r *Results) ExportJSON(filename string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(r.pages)
}

// ExportCSV exports results to CSV file
func (r *Results) ExportCSV(filename string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"URL", "Title", "Description", "Links Count", "Response Time (ms)", "Success", "Error"}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write rows
	for _, page := range r.pages {
		row := []string{
			page.URL,
			page.Title,
			page.Description,
			fmt.Sprintf("%d", len(page.Links)),
			fmt.Sprintf("%d", page.ResponseTime.Milliseconds()),
			fmt.Sprintf("%t", page.Success),
			page.Error,
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// ExportLinksCSV exports all links found to a separate CSV file
func (r *Results) ExportLinksCSV(filename string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"Source URL", "Found Link", "Link Depth"}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write rows - one row per link found
	for _, page := range r.pages {
		if !page.Success {
			continue
		}
		for _, link := range page.Links {
			row := []string{
				page.URL,
				link,
				"", // Depth could be calculated if needed
			}
			if err := writer.Write(row); err != nil {
				return err
			}
		}
	}

	return nil
}
