package collector

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/thd3r/hot-news-collector/internal/models"
)

// NewsCollector manages news collection from multiple sources
type NewsCollector struct {
	Config       *models.Config
	HTTPClient   *http.Client
	Sources      map[string]models.Source
	SeenArticles map[string]bool
	RateLimiter  chan struct{}
	mu           sync.RWMutex
}

// generateArticleID creates a unique ID for deduplication
func (nc *NewsCollector) generateArticleID(title, url string) string {
	combined := strings.ToLower(title + url)
	hash := md5.Sum([]byte(combined))
	return fmt.Sprintf("%x", hash)[:16]
}

// isArticleSeen checks if article was already processed
func (nc *NewsCollector) isArticleSeen(id string) bool {
	nc.mu.RLock()
	defer nc.mu.RUnlock()
	return nc.SeenArticles[id]
}

// markArticleSeen marks article as processed
func (nc *NewsCollector) markArticleSeen(id string) {
	nc.mu.Lock()
	defer nc.mu.Unlock()
	nc.SeenArticles[id] = true
}

// extractKeywords extracts relevant keywords from text
func (nc *NewsCollector) extractKeywords(text string) []string {
	text = strings.ToLower(text)

	// Common tech keywords to extract
	techKeywords := []string{
		"ai", "artificial intelligence", "machine learning", "blockchain", "cryptocurrency",
		"cybersecurity", "startup", "funding", "ipo", "acquisition", "breakthrough",
		"innovation", "technology", "software", "hardware", "cloud", "saas", "api",
	}

	var found []string
	for _, keyword := range techKeywords {
		if strings.Contains(text, keyword) {
			found = append(found, keyword)
		}
	}

	return found
}

// rateLimitedRequest performs rate-limited HTTP request
func (nc *NewsCollector) RateLimitedRequest(req *http.Request) (*http.Response, error) {
	nc.RateLimiter <- struct{}{}   // Acquire slot
	defer func() { <-nc.RateLimiter }() // Release slot

	time.Sleep(100 * time.Millisecond) // Small delay between requests
	return nc.HTTPClient.Do(req)
}

// CollectAllNews collects and filters news from all sources
func (nc *NewsCollector) CollectAllNews() ([]models.NewsArticle, error) {
	var allArticles []models.NewsArticle
	var wg sync.WaitGroup

	articlesChan := make(chan []models.NewsArticle, 10)

	categories := nc.CategoryDetector.GetAllCategories()

	// Collect from NewsAPI concurrently
	if nc.NewsAPIKey != "" {
		for _, category := range categories {
			wg.Add(1)
			go func(cat string) {
				defer wg.Done()
				articles, err := nc.GetNewsAPIArticles(cat)
				if err != nil {
					fmt.Printf("Error fetching NewsAPI %s: %v\n", cat, err)
					return
				}
				articlesChan <- articles
			}(category)
		}
	}

	// Collect from Hacker News
	wg.Add(1)
	go func() {
		defer wg.Done()
		articles, err := nc.GetHackerNewsStories()
		if err != nil {
			fmt.Printf("Error fetching Hacker News: %v\n", err)
			return
		}
		articlesChan <- articles
	}()

	// Collect from Reddit
	wg.Add(1)
	go func() {
		defer wg.Done()
		articles, err := nc.GetRedditTechNews()
		if err != nil {
			fmt.Printf("Error fetching Reddit: %v\n", err)
			return
		}
		articlesChan <- articles
	}()

	// Close channel when all collections complete
	go func() {
		wg.Wait()
		close(articlesChan)
	}()

	// Collect all results
	for articles := range articlesChan {
		allArticles = append(allArticles, articles...)
	}

	// Sort by score (highest first)
	sort.Slice(allArticles, func(i, j int) bool {
		return allArticles[i].Score > allArticles[j].Score
	})

	return allArticles, nil
}
