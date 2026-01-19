package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"gocrawler/crawler"
	"gocrawler/storage"
	"gocrawler/web"
)

func main() {
	// Parse command-line flags
	startURL := flag.String("url", "https://golang.org", "Starting URL to crawl")
	maxDepth := flag.Int("depth", 2, "Maximum crawl depth")
	workers := flag.Int("workers", 10, "Number of concurrent workers")
	rateLimit := flag.Int("rate", 10, "Requests per second limit")
	webPort := flag.Int("port", 8080, "Web dashboard port")
	flag.Parse()

	fmt.Printf(`
╔═══════════════════════════════════════════════════════════╗
║           Go Concurrent Web Crawler v1.0                  ║
║  Demonstrating: Goroutines, Channels, Context & More     ║
╚═══════════════════════════════════════════════════════════╝

Configuration:
  • Start URL:     %s
  • Max Depth:     %d
  • Workers:       %d (concurrent goroutines)
  • Rate Limit:    %d req/sec
  • Dashboard:     http://localhost:%d

Press Ctrl+C to stop crawling...

`, *startURL, *maxDepth, *workers, *rateLimit, *webPort)

	// Create results storage
	results := storage.NewResults()

	// Create crawler with context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := crawler.New(*workers, *rateLimit, *maxDepth, results)

	// Start web dashboard in goroutine
	srv := web.NewServer(*webPort, results)
	go func() {
		if err := srv.Start(); err != nil {
			log.Printf("Web server error: %v", err)
		}
	}()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start crawling in goroutine
	done := make(chan bool)
	go func() {
		c.Crawl(ctx, *startURL)
		done <- true
	}()

	// Wait for completion or interruption
	select {
	case <-sigChan:
		fmt.Println("\n\n🛑 Interrupt received, stopping crawler...")
		cancel()
		<-done // Wait for crawler to finish
	case <-done:
		fmt.Println("\n\n✅ Crawling completed!")
	}

	// Print final statistics
	printStats(results)

	// Export results
	if err := results.ExportJSON("crawl_results.json"); err != nil {
		log.Printf("Error exporting JSON: %v", err)
	}
	if err := results.ExportCSV("crawl_results.csv"); err != nil {
		log.Printf("Error exporting CSV: %v", err)
	}
	if err := results.ExportLinksCSV("crawl_links.csv"); err != nil {
		log.Printf("Error exporting links CSV: %v", err)
	}

	fmt.Println("\n📊 Results exported:")
	fmt.Println("   • crawl_results.json - All page data")
	fmt.Println("   • crawl_results.csv - Page summary")
	fmt.Println("   • crawl_links.csv - All links found (easier to read)")
	fmt.Println("🌐 Dashboard available at http://localhost:8080")
	fmt.Println("\nPress Ctrl+C again to exit dashboard...")

	// Keep dashboard running
	<-sigChan
	fmt.Println("\n👋 Goodbye!")
}

func printStats(results *storage.Results) {
	stats := results.GetStats()

	fmt.Printf(`
╔═══════════════════════════════════════════════════════════╗
║                    Crawling Statistics                    ║
╚═══════════════════════════════════════════════════════════╝

📄 Pages Crawled:     %d
🔗 Unique Links:      %d
⏱️  Average Time:      %.2f ms
✅ Successful:        %d
❌ Failed:            %d
⚡ Crawl Duration:    %s

`, stats.TotalPages, stats.UniqueLinks, stats.AvgResponseTime,
		stats.SuccessCount, stats.FailCount, stats.Duration)
}
