package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// NewsArticle represents a single news article
type NewsArticle struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	Source      string    `json:"source"`
	PublishedAt time.Time `json:"published_at"`
	Category    string    `json:"category"`
	Score       int       `json:"score"`
	Keywords    []string  `json:"keywords"`
}

// NewsAPIResponse represents the response from NewsAPI
type NewsAPIResponse struct {
	Status       string `json:"status"`
	TotalResults int    `json:"totalResults"`
	Articles     []struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		URL         string `json:"url"`
		Source      struct {
			Name string `json:"name"`
		} `json:"source"`
		PublishedAt string `json:"publishedAt"`
	} `json:"articles"`
}

// HackerNewsItem represents a Hacker News item
type HackerNewsItem struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Score       int    `json:"score"`
	Time        int64  `json:"time"`
	Descendants int    `json:"descendants"`
}

// CategoryFilter defines advanced filtering rules for each category
type CategoryFilter struct {
	PrimaryKeywords   []string
	SecondaryKeywords []string
	ExcludeKeywords   []string
	MinScore          int
	WeightMultiplier  float64
}

// NewsCollector manages news collection from multiple sources
type NewsCollector struct {
	NewsAPIKey      string
	HTTPClient      *http.Client
	CategoryFilters map[string]CategoryFilter
	SeenArticles    map[string]bool
	RateLimiter     chan struct{}
	mu              sync.RWMutex
}

// NewNewsCollector creates a new news collector with improved filtering
func NewNewsCollector(apiKey string) *NewsCollector {
	// Define advanced category filters
	categoryFilters := map[string]CategoryFilter{
		"ai": {
			PrimaryKeywords: []string{
				"artificial intelligence", "machine learning", "deep learning", "neural network",
				"GPT", "ChatGPT", "OpenAI", "AI model", "generative AI", "LLM", "transformer",
				"computer vision", "natural language processing", "NLP", "reinforcement learning",
			},
			SecondaryKeywords: []string{
				"algorithm", "automation", "robot", "cognitive", "intelligent", "smart system",
				"predictive", "recommendation", "pattern recognition", "data mining",
			},
			ExcludeKeywords: []string{
				"game", "movie", "entertainment", "sport", "weather", "fashion",
			},
			MinScore:         25,
			WeightMultiplier: 1.5,
		},
		"tech": {
			PrimaryKeywords: []string{
				"startup", "technology", "software", "hardware", "programming", "developer",
				"innovation", "digital transformation", "cloud computing", "SaaS", "API",
				"mobile app", "web development", "IoT", "5G", "quantum computing",
			},
			SecondaryKeywords: []string{
				"tech company", "silicon valley", "venture capital", "funding", "IPO",
				"platform", "framework", "open source", "github", "coding",
			},
			ExcludeKeywords: []string{
				"celebrity", "politics", "entertainment", "sport", "weather",
			},
			MinScore:         20,
			WeightMultiplier: 1.2,
		},
		"digital": {
			PrimaryKeywords: []string{
				"cryptocurrency", "blockchain", "bitcoin", "ethereum", "fintech", "digital payment",
				"NFT", "DeFi", "digital currency", "crypto", "digital wallet", "trading",
				"financial technology", "digital banking", "e-commerce", "digital economy",
			},
			SecondaryKeywords: []string{
				"investment", "market", "finance", "trading platform", "exchange",
				"digital transformation", "online payment", "mobile payment",
			},
			ExcludeKeywords: []string{
				"celebrity", "entertainment", "sport", "weather", "politics unrelated",
			},
			MinScore:         22,
			WeightMultiplier: 1.3,
		},
		"hacking": {
			PrimaryKeywords: []string{
				"cybersecurity", "data breach", "hacking", "cyber attack", "malware",
				"ransomware", "phishing", "vulnerability", "security breach", "cyber threat",
				"information security", "network security", "data leak", "exploit",
			},
			SecondaryKeywords: []string{
				"security", "privacy", "encryption", "firewall", "antivirus",
				"penetration testing", "ethical hacking", "cyber defense",
			},
			ExcludeKeywords: []string{
				"movie", "game", "entertainment", "sport", "weather",
			},
			MinScore:         30,
			WeightMultiplier: 1.4,
		},
	}

	return &NewsCollector{
		NewsAPIKey:      apiKey,
		CategoryFilters: categoryFilters,
		SeenArticles:    make(map[string]bool),
		RateLimiter:     make(chan struct{}, 5), // Max 5 concurrent requests
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
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

// advancedCategoryDetection uses improved algorithm to detect article category
func (nc *NewsCollector) advancedCategoryDetection(title, description string) (string, int) {
	content := strings.ToLower(title + " " + description)

	bestCategory := ""
	bestScore := 0

	for category, filter := range nc.CategoryFilters {
		score := 0

		// Check for exclusion keywords first
		excluded := false
		for _, keyword := range filter.ExcludeKeywords {
			if strings.Contains(content, strings.ToLower(keyword)) {
				excluded = true
				break
			}
		}

		if excluded {
			continue
		}

		// Primary keywords (high weight)
		for _, keyword := range filter.PrimaryKeywords {
			if strings.Contains(content, strings.ToLower(keyword)) {
				score += 20
				// Bonus if keyword appears in title
				if strings.Contains(strings.ToLower(title), strings.ToLower(keyword)) {
					score += 10
				}
			}
		}

		// Secondary keywords (medium weight)
		for _, keyword := range filter.SecondaryKeywords {
			if strings.Contains(content, strings.ToLower(keyword)) {
				score += 8
			}
		}

		// Apply category multiplier
		score = int(float64(score) * filter.WeightMultiplier)

		// Check minimum score threshold
		if score >= filter.MinScore && score > bestScore {
			bestScore = score
			bestCategory = category
		}
	}

	return bestCategory, bestScore
}

// rateLimitedRequest performs rate-limited HTTP request
func (nc *NewsCollector) rateLimitedRequest(req *http.Request) (*http.Response, error) {
	nc.RateLimiter <- struct{}{}        // Acquire slot
	defer func() { <-nc.RateLimiter }() // Release slot

	time.Sleep(100 * time.Millisecond) // Small delay between requests
	return nc.HTTPClient.Do(req)
}

// GetNewsAPIArticles fetches articles from NewsAPI with improved filtering
func (nc *NewsCollector) GetNewsAPIArticles(category string) ([]NewsArticle, error) {
	if nc.NewsAPIKey == "" {
		return nil, fmt.Errorf("NewsAPI key not provided")
	}

	baseURL := "https://newsapi.org/v2/everything"

	// Improved search queries based on category filters
	queries := map[string]string{
		"ai":      "\"artificial intelligence\" OR \"machine learning\" OR \"deep learning\" OR ChatGPT OR OpenAI OR \"AI model\"",
		"tech":    "\"startup funding\" OR \"tech company\" OR \"software development\" OR \"cloud computing\" OR \"digital transformation\"",
		"digital": "cryptocurrency OR blockchain OR bitcoin OR fintech OR \"digital payment\" OR NFT OR DeFi",
		"hacking": "\"cyber attack\" OR \"data breach\" OR ransomware OR \"cybersecurity\" OR \"security vulnerability\"",
	}

	query, exists := queries[category]
	if !exists {
		return nil, fmt.Errorf("unknown category: %s", category)
	}

	params := url.Values{}
	params.Add("q", query)
	params.Add("language", "en")
	params.Add("sortBy", "publishedAt")
	params.Add("pageSize", "50")                                          // Increased to get more candidates
	params.Add("from", time.Now().AddDate(0, 0, -2).Format("2006-01-02")) // Last 2 days

	reqURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("X-API-Key", nc.NewsAPIKey)
	req.Header.Set("User-Agent", "GoNewsCollector/2.0")

	resp, err := nc.rateLimitedRequest(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var newsResp NewsAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&newsResp); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}

	var articles []NewsArticle
	for _, article := range newsResp.Articles {
		// Skip articles with missing essential fields
		if article.Title == "" || article.URL == "" {
			continue
		}

		// Generate unique ID for deduplication
		articleID := nc.generateArticleID(article.Title, article.URL)
		if nc.isArticleSeen(articleID) {
			continue
		}

		// Advanced category detection
		detectedCategory, score := nc.advancedCategoryDetection(article.Title, article.Description)
		if detectedCategory == "" || score < nc.CategoryFilters[category].MinScore {
			continue // Skip articles that don't meet category criteria
		}

		// Parse published date with multiple format support
		var publishedAt time.Time
		timeFormats := []string{
			"2006-01-02T15:04:05Z",
			"2006-01-02T15:04:05.000Z",
			"2006-01-02T15:04:05-07:00",
		}

		for _, format := range timeFormats {
			if publishedAt, err = time.Parse(format, article.PublishedAt); err == nil {
				break
			}
		}

		if publishedAt.IsZero() {
			publishedAt = time.Now()
		}

		newsArticle := NewsArticle{
			ID:          articleID,
			Title:       strings.TrimSpace(article.Title),
			Description: strings.TrimSpace(article.Description),
			URL:         article.URL,
			Source:      article.Source.Name,
			PublishedAt: publishedAt,
			Category:    detectedCategory,
			Score:       score,
			Keywords:    nc.extractKeywords(article.Title + " " + article.Description),
		}

		articles = append(articles, newsArticle)
		nc.markArticleSeen(articleID)
	}

	return articles, nil
}

// GetHackerNewsStories fetches tech stories from Hacker News with improved filtering
func (nc *NewsCollector) GetHackerNewsStories() ([]NewsArticle, error) {
	resp, err := nc.HTTPClient.Get("https://hacker-news.firebaseio.com/v0/topstories.json")
	if err != nil {
		return nil, fmt.Errorf("fetching top stories: %w", err)
	}
	defer resp.Body.Close()

	var storyIDs []int
	if err := json.NewDecoder(resp.Body).Decode(&storyIDs); err != nil {
		return nil, fmt.Errorf("decoding story IDs: %w", err)
	}

	var articles []NewsArticle
	var wg sync.WaitGroup
	articlesChan := make(chan NewsArticle, 30)

	// Process top 30 stories concurrently
	for i, id := range storyIDs {
		if i >= 30 {
			break
		}

		wg.Add(1)
		go func(storyID int) {
			defer wg.Done()

			itemURL := fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", storyID)
			resp, err := nc.HTTPClient.Get(itemURL)
			if err != nil {
				log.Printf("Error fetching story %d: %v", storyID, err)
				return
			}
			defer resp.Body.Close()

			var item HackerNewsItem
			if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
				log.Printf("Error decoding story %d: %v", storyID, err)
				return
			}

			if item.Title == "" {
				return
			}

			// Generate unique ID
			articleID := nc.generateArticleID(item.Title, item.URL)
			if nc.isArticleSeen(articleID) {
				return
			}

			// Advanced category detection
			category, score := nc.advancedCategoryDetection(item.Title, "")
			if category == "" || score < 15 { // Lower threshold for HN
				return
			}

			url := item.URL
			if url == "" {
				url = fmt.Sprintf("https://news.ycombinator.com/item?id=%d", item.ID)
			}

			article := NewsArticle{
				ID:          articleID,
				Title:       strings.TrimSpace(item.Title),
				URL:         url,
				Source:      "Hacker News",
				PublishedAt: time.Unix(item.Time, 0),
				Category:    category,
				Score:       score + (item.Score / 10), // Incorporate HN score
				Description: fmt.Sprintf("HN Score: %d, Comments: %d", item.Score, item.Descendants),
				Keywords:    nc.extractKeywords(item.Title),
			}

			articlesChan <- article
		}(id)
	}

	// Close channel when all goroutines complete
	go func() {
		wg.Wait()
		close(articlesChan)
	}()

	// Collect results
	for article := range articlesChan {
		articles = append(articles, article)
		nc.markArticleSeen(article.ID)
	}

	return articles, nil
}

// GetRedditTechNews fetches tech news from Reddit with improved filtering
func (nc *NewsCollector) GetRedditTechNews() ([]NewsArticle, error) {
	subreddits := []string{"technology", "artificial", "cybersecurity", "programming", "startups", "MachineLearning"}
	var allArticles []NewsArticle

	for _, subreddit := range subreddits {
		url := fmt.Sprintf("https://www.reddit.com/r/%s/hot.json?limit=15", subreddit)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Printf("Error creating Reddit request for %s: %v", subreddit, err)
			continue
		}

		req.Header.Set("User-Agent", "GoNewsCollector/2.0")

		resp, err := nc.rateLimitedRequest(req)
		if err != nil {
			log.Printf("Error fetching Reddit %s: %v", subreddit, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("Reddit API returned status %d for r/%s", resp.StatusCode, subreddit)
			continue
		}

		var redditResp struct {
			Data struct {
				Children []struct {
					Data struct {
						Title      string  `json:"title"`
						URL        string  `json:"url"`
						Score      int     `json:"score"`
						CreatedUTC float64 `json:"created_utc"`
						Selftext   string  `json:"selftext"`
					} `json:"data"`
				} `json:"children"`
			} `json:"data"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&redditResp); err != nil {
			log.Printf("Error decoding Reddit response for %s: %v", subreddit, err)
			continue
		}

		for _, child := range redditResp.Data.Children {
			if child.Data.Title == "" || child.Data.Score < 50 { // Quality threshold
				continue
			}

			// Skip self posts without external URLs
			if !strings.HasPrefix(child.Data.URL, "http") {
				continue
			}

			// Generate unique ID
			articleID := nc.generateArticleID(child.Data.Title, child.Data.URL)
			if nc.isArticleSeen(articleID) {
				continue
			}

			// Advanced category detection
			category, score := nc.advancedCategoryDetection(child.Data.Title, child.Data.Selftext)
			if category == "" || score < 18 {
				continue
			}

			article := NewsArticle{
				ID:          articleID,
				Title:       strings.TrimSpace(child.Data.Title),
				URL:         child.Data.URL,
				Source:      fmt.Sprintf("Reddit r/%s", subreddit),
				PublishedAt: time.Unix(int64(child.Data.CreatedUTC), 0),
				Category:    category,
				Score:       score + (child.Data.Score / 50), // Incorporate Reddit score
				Description: fmt.Sprintf("Reddit Score: %d", child.Data.Score),
				Keywords:    nc.extractKeywords(child.Data.Title + " " + child.Data.Selftext),
			}

			allArticles = append(allArticles, article)
			nc.markArticleSeen(articleID)
		}

		// Small delay between subreddit requests
		time.Sleep(200 * time.Millisecond)
	}

	return allArticles, nil
}

// CollectAllNews collects and filters news from all sources
func (nc *NewsCollector) CollectAllNews() ([]NewsArticle, error) {
	var allArticles []NewsArticle
	var wg sync.WaitGroup
	articlesChan := make(chan []NewsArticle, 10)

	categories := []string{"ai", "tech", "digital", "hacking"}

	// Collect from NewsAPI concurrently
	if nc.NewsAPIKey != "" {
		for _, category := range categories {
			wg.Add(1)
			go func(cat string) {
				defer wg.Done()
				articles, err := nc.GetNewsAPIArticles(cat)
				if err != nil {
					log.Printf("Error fetching NewsAPI %s: %v", cat, err)
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
			log.Printf("Error fetching Hacker News: %v", err)
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
			log.Printf("Error fetching Reddit: %v", err)
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

// ImprovedHotNewsDetector with enhanced scoring algorithm
type ImprovedHotNewsDetector struct {
	ViralKeywords   []string
	TrendingTopics  []string
	SourceWeights   map[string]float64
	CategoryWeights map[string]float64
}

// NewImprovedHotNewsDetector creates enhanced hot news detector
func NewImprovedHotNewsDetector() *ImprovedHotNewsDetector {
	return &ImprovedHotNewsDetector{
		ViralKeywords: []string{
			"breaking", "urgent", "exclusive", "leaked", "exposed", "massive", "billion",
			"breakthrough", "revolutionary", "first-ever", "record-breaking", "unprecedented",
			"crisis", "emergency", "alert", "warning", "critical", "major", "huge",
			"shocking", "stunning", "dramatic", "historic", "landmark",
		},
		TrendingTopics: []string{
			"ChatGPT", "OpenAI", "Google Bard", "Microsoft", "Apple", "Tesla", "Meta",
			"cybersecurity", "data breach", "AI regulation", "cryptocurrency crash",
			"startup funding", "IPO", "acquisition", "layoffs", "hiring freeze",
		},
		SourceWeights: map[string]float64{
			"TechCrunch":   1.5,
			"Wired":        1.4,
			"Ars Technica": 1.4,
			"The Verge":    1.3,
			"Reuters":      1.3,
			"Hacker News":  1.2,
			"Reddit":       1.0,
		},
		CategoryWeights: map[string]float64{
			"hacking": 1.4,
			"ai":      1.3,
			"digital": 1.2,
			"tech":    1.1,
		},
	}
}

// CalculateEnhancedHotScore calculates comprehensive hotness score
func (hnd *ImprovedHotNewsDetector) CalculateEnhancedHotScore(article NewsArticle) int {
	score := float64(article.Score) // Start with existing score

	titleLower := strings.ToLower(article.Title)
	descLower := strings.ToLower(article.Description)
	combined := titleLower + " " + descLower

	// Viral keywords bonus (exponential for multiple matches)
	viralMatches := 0
	for _, keyword := range hnd.ViralKeywords {
		if strings.Contains(combined, keyword) {
			viralMatches++
		}
	}
	if viralMatches > 0 {
		score += float64(viralMatches*viralMatches) * 10
	}

	// Trending topics bonus
	for _, topic := range hnd.TrendingTopics {
		if strings.Contains(combined, strings.ToLower(topic)) {
			score += 15
			// Extra bonus if in title
			if strings.Contains(titleLower, strings.ToLower(topic)) {
				score += 10
			}
		}
	}

	// Recency bonus (stronger decay)
	hoursOld := time.Since(article.PublishedAt).Hours()
	if hoursOld < 1 {
		score *= 1.8
	} else if hoursOld < 3 {
		score *= 1.5
	} else if hoursOld < 6 {
		score *= 1.3
	} else if hoursOld < 12 {
		score *= 1.1
	} else if hoursOld > 48 {
		score *= 0.7
	}

	// Source credibility multiplier
	sourceWeight := hnd.SourceWeights[article.Source]
	if sourceWeight == 0 {
		sourceWeight = 1.0
	}
	score *= sourceWeight

	// Category importance multiplier
	categoryWeight := hnd.CategoryWeights[article.Category]
	if categoryWeight == 0 {
		categoryWeight = 1.0
	}
	score *= categoryWeight

	// Title quality bonus
	titleWords := strings.Fields(article.Title)
	if len(titleWords) >= 5 && len(titleWords) <= 15 { // Optimal title length
		score += 5
	}

	// Number/statistics bonus
	numberRegex := regexp.MustCompile(`\b\d+(\.\d+)?[BbMmKk]?\b`)
	if numberRegex.MatchString(article.Title) {
		score += 8
	}

	// Penalty for very short descriptions
	if len(article.Description) < 50 {
		score *= 0.9
	}

	return int(score)
}

// GetHottestNews returns the hottest news with enhanced filtering
func (hnd *ImprovedHotNewsDetector) GetHottestNews(articles []NewsArticle, limit int) []NewsArticle {
	type ScoredArticle struct {
		Article  NewsArticle
		HotScore int
	}

	var scored []ScoredArticle

	for _, article := range articles {
		hotScore := hnd.CalculateEnhancedHotScore(article)
		if hotScore >= 35 { // Higher threshold for quality
			scored = append(scored, ScoredArticle{
				Article:  article,
				HotScore: hotScore,
			})
		}
	}

	// Sort by hot score
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].HotScore > scored[j].HotScore
	})

	// Return top articles with updated scores
	var hotArticles []NewsArticle
	for i, item := range scored {
		if i >= limit {
			break
		}
		article := item.Article
		article.Score = item.HotScore // Update with hot score
		hotArticles = append(hotArticles, article)
	}

	return hotArticles
}

// Enhanced Twitter post generation with better templates
func generateEnhancedTwitterPosts(articles []NewsArticle) []map[string]interface{} {
	templates := map[string][]string{
		"ai": {
			"🤖 AI BREAKING: %s\n\nThe future is NOW! This changes everything in tech.\n\n🔗 %s\n\n%s",
			"⚡ MASSIVE AI NEWS: %s\n\nMind = BLOWN 🤯 The AI revolution accelerates!\n\n📖 %s\n\n%s",
			"🚨 AI ALERT: %s\n\nThis is HUGE for artificial intelligence! 🧠⚡\n\n🔗 %s\n\n%s",
		},
		"tech": {
			"🔥 TECH BOMB: %s\n\nSilicon Valley is BUZZING! This is massive! 💥\n\n📖 %s\n\n%s",
			"🚀 STARTUP ALERT: %s\n\nThe tech world just shifted! Game changer! 🎯\n\n🔗 %s\n\n%s",
			"💡 INNOVATION BLAST: %s\n\nTech disruption in real-time! Don't miss this! ⚡\n\n📖 %s\n\n%s",
		},
		"hacking": {
			"🚨 CYBER CRISIS: %s\n\nMajor security alert! Check your systems NOW! 🔒\n\n🔗 %s\n\n%s",
			"⚠️ BREACH ALERT: %s\n\nCybersecurity nightmare unfolding! Stay protected! 🛡️\n\n📖 %s\n\n%s",
			"🔴 SECURITY EMERGENCY: %s\n\nHackers are evolving! Critical update needed! 🚨\n\n🔗 %s\n\n%s",
		},
		"digital": {
			"💰 CRYPTO EXPLOSION: %s\n\nThe digital economy is on fire! 🔥💎\n\n📖 %s\n\n%s",
			"🚀 BLOCKCHAIN BOOM: %s\n\nFintech revolution happening NOW! Don't miss out! 💸\n\n🔗 %s\n\n%s",
			"💎 DIGITAL GOLD RUSH: %s\n\nThe future of money is here! Game changer! ⚡\n\n📖 %s\n\n%s",
		},
	}

	var posts []map[string]interface{}

	for i, article := range articles {
		if i >= 10 { // Limit to top 10 articles
			break
		}

		categoryTemplates := templates[article.Category]
		if len(categoryTemplates) == 0 {
			continue
		}

		// Select template based on article index
		template := categoryTemplates[i%len(categoryTemplates)]

		// Generate hashtags based on category and keywords
		hashtags := generateHashtags(article.Category, article.Keywords)

		// Format the post
		post := fmt.Sprintf(template,
			truncateTitle(article.Title, 80),
			article.URL,
			hashtags)

		// Ensure post is within Twitter character limit
		if len(post) > 280 {
			post = truncatePost(post, 280)
		}

		posts = append(posts, map[string]interface{}{
			"content":    post,
			"article_id": article.ID,
			"category":   article.Category,
			"score":      article.Score,
			"source":     article.Source,
			"scheduled":  false,
		})
	}

	return posts
}

// generateHashtags creates relevant hashtags for the post
func generateHashtags(category string, keywords []string) string {
	categoryHashtags := map[string][]string{
		"ai":      {"#AI", "#MachineLearning", "#TechNews", "#Innovation", "#FutureOfWork"},
		"tech":    {"#TechNews", "#Startup", "#Innovation", "#SiliconValley", "#Technology"},
		"hacking": {"#Cybersecurity", "#InfoSec", "#DataBreach", "#CyberAttack", "#Security"},
		"digital": {"#Crypto", "#Blockchain", "#Fintech", "#DigitalCurrency", "#Web3"},
	}

	var hashtags []string

	// Add category-specific hashtags
	if categoryTags, exists := categoryHashtags[category]; exists {
		hashtags = append(hashtags, categoryTags[:3]...) // Take first 3
	}

	// Add keyword-based hashtags
	keywordHashtags := map[string]string{
		"openai":   "#OpenAI",
		"chatgpt":  "#ChatGPT",
		"bitcoin":  "#Bitcoin",
		"ethereum": "#Ethereum",
		"startup":  "#Startup",
		"funding":  "#Funding",
	}

	for _, keyword := range keywords {
		if hashtag, exists := keywordHashtags[strings.ToLower(keyword)]; exists {
			hashtags = append(hashtags, hashtag)
			if len(hashtags) >= 5 { // Limit hashtags
				break
			}
		}
	}

	return strings.Join(hashtags, " ")
}

// truncateTitle truncates title to specified length
func truncateTitle(title string, maxLength int) string {
	if len(title) <= maxLength {
		return title
	}

	truncated := title[:maxLength-3]
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > 0 {
		truncated = truncated[:lastSpace]
	}

	return truncated + "..."
}

// truncatePost truncates entire post to fit character limit
func truncatePost(post string, maxLength int) string {
	if len(post) <= maxLength {
		return post
	}

	lines := strings.Split(post, "\n")
	var result []string
	currentLength := 0

	for _, line := range lines {
		if currentLength+len(line)+1 > maxLength {
			break
		}
		result = append(result, line)
		currentLength += len(line) + 1
	}

	return strings.Join(result, "\n")
}

// NewsReporter generates comprehensive news reports
type NewsReporter struct {
	Articles []NewsArticle
}

// NewNewsReporter creates a new news reporter
func NewNewsReporter(articles []NewsArticle) *NewsReporter {
	return &NewsReporter{Articles: articles}
}

// GenerateHTMLReport creates a beautiful HTML report
func (nr *NewsReporter) GenerateHTMLReport() string {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Tech News Report - ` + time.Now().Format("January 2, 2006") + `</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { 
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            line-height: 1.6; 
            color: #333; 
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
        }
        .container { 
            max-width: 1200px; 
            margin: 0 auto; 
            padding: 20px;
        }
        .header {
            text-align: center;
            color: white;
            margin-bottom: 40px;
        }
        .header h1 {
            font-size: 3em;
            margin-bottom: 10px;
            text-shadow: 2px 2px 4px rgba(0,0,0,0.3);
        }
        .header p {
            font-size: 1.2em;
            opacity: 0.9;
        }
        .stats {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin-bottom: 40px;
        }
        .stat-card {
            background: rgba(255,255,255,0.1);
            backdrop-filter: blur(10px);
            padding: 20px;
            border-radius: 15px;
            text-align: center;
            color: white;
            border: 1px solid rgba(255,255,255,0.2);
        }
        .stat-number {
            font-size: 2.5em;
            font-weight: bold;
            display: block;
        }
        .category-section {
            margin-bottom: 40px;
        }
        .category-header {
            background: rgba(255,255,255,0.95);
            padding: 20px;
            border-radius: 15px 15px 0 0;
            border-left: 5px solid;
        }
        .category-ai { border-left-color: #ff6b6b; }
        .category-tech { border-left-color: #4ecdc4; }
        .category-digital { border-left-color: #45b7d1; }
        .category-hacking { border-left-color: #f9ca24; }
        
        .category-title {
            font-size: 1.8em;
            font-weight: bold;
            color: #2c3e50;
            display: flex;
            align-items: center;
        }
        .category-icon {
            margin-right: 10px;
            font-size: 1.2em;
        }
        .articles-grid {
            display: grid;
            gap: 20px;
            background: rgba(255,255,255,0.95);
            padding: 20px;
            border-radius: 0 0 15px 15px;
        }
        .article-card {
            background: white;
            padding: 20px;
            border-radius: 10px;
            box-shadow: 0 4px 6px rgba(0,0,0,0.1);
            border-left: 4px solid;
            transition: transform 0.2s, box-shadow 0.2s;
        }
        .article-card:hover {
            transform: translateY(-2px);
            box-shadow: 0 8px 15px rgba(0,0,0,0.2);
        }
        .article-ai { border-left-color: #ff6b6b; }
        .article-tech { border-left-color: #4ecdc4; }
        .article-digital { border-left-color: #45b7d1; }
        .article-hacking { border-left-color: #f9ca24; }
        
        .article-title {
            font-size: 1.3em;
            font-weight: bold;
            margin-bottom: 10px;
            color: #2c3e50;
        }
        .article-title a {
            color: inherit;
            text-decoration: none;
        }
        .article-title a:hover {
            color: #3498db;
        }
        .article-meta {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 10px;
            font-size: 0.9em;
            color: #7f8c8d;
        }
        .article-description {
            color: #555;
            margin-bottom: 10px;
        }
        .keywords {
            display: flex;
            flex-wrap: wrap;
            gap: 5px;
        }
        .keyword-tag {
            background: #ecf0f1;
            padding: 3px 8px;
            border-radius: 12px;
            font-size: 0.8em;
            color: #555;
        }
        .hot-score {
            background: linear-gradient(45deg, #ff6b6b, #ee5a6f);
            color: white;
            padding: 5px 10px;
            border-radius: 15px;
            font-weight: bold;
            font-size: 0.9em;
        }
        .footer {
            text-align: center;
            color: rgba(255,255,255,0.8);
            margin-top: 40px;
            padding: 20px;
        }
        @media (max-width: 768px) {
            .container { padding: 10px; }
            .header h1 { font-size: 2em; }
            .articles-grid { grid-template-columns: 1fr; }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🚀 Tech News Report</h1>
            <p>Latest Technology News • ` + time.Now().Format("Monday, January 2, 2006") + `</p>
        </div>`

	// Add statistics
	categoryStats := make(map[string]int)
	totalScore := 0
	for _, article := range nr.Articles {
		categoryStats[article.Category]++
		totalScore += article.Score
	}

	html += `
        <div class="stats">
            <div class="stat-card">
                <span class="stat-number">` + fmt.Sprintf("%d", len(nr.Articles)) + `</span>
                <div>Total Articles</div>
            </div>
            <div class="stat-card">
                <span class="stat-number">` + fmt.Sprintf("%d", len(categoryStats)) + `</span>
                <div>Categories</div>
            </div>
            <div class="stat-card">
                <span class="stat-number">` + fmt.Sprintf("%d", totalScore/len(nr.Articles)) + `</span>
                <div>Avg. Hot Score</div>
            </div>
            <div class="stat-card">
                <span class="stat-number">` + fmt.Sprintf("%.1f", time.Since(nr.Articles[0].PublishedAt).Hours()) + `h</span>
                <div>Latest News</div>
            </div>
        </div>`

	// Group articles by category
	categorizedArticles := make(map[string][]NewsArticle)
	for _, article := range nr.Articles {
		categorizedArticles[article.Category] = append(categorizedArticles[article.Category], article)
	}

	// Category icons and names
	categoryInfo := map[string]struct {
		icon string
		name string
	}{
		"ai":      {"🤖", "Artificial Intelligence"},
		"tech":    {"💻", "Technology"},
		"digital": {"💰", "Digital & Fintech"},
		"hacking": {"🔒", "Cybersecurity"},
	}

	// Generate sections for each category
	for category, articles := range categorizedArticles {
		if len(articles) == 0 {
			continue
		}

		info := categoryInfo[category]
		html += fmt.Sprintf(`
        <div class="category-section">
            <div class="category-header category-%s">
                <div class="category-title">
                    <span class="category-icon">%s</span>
                    %s (%d articles)
                </div>
            </div>
            <div class="articles-grid">`, category, info.icon, info.name, len(articles))

		for _, article := range articles {
			timeAgo := formatTimeAgo(article.PublishedAt)
			keywordTags := ""
			for _, keyword := range article.Keywords {
				if len(keyword) > 0 {
					keywordTags += fmt.Sprintf(`<span class="keyword-tag">%s</span>`, keyword)
				}
			}

			html += fmt.Sprintf(`
                <div class="article-card article-%s">
                    <div class="article-title">
                        <a href="%s" target="_blank">%s</a>
                    </div>
                    <div class="article-meta">
                        <span>%s • %s</span>
                        <span class="hot-score">🔥 %d</span>
                    </div>
                    <div class="article-description">%s</div>
                    <div class="keywords">%s</div>
                </div>`,
				category,
				article.URL,
				article.Title,
				article.Source,
				timeAgo,
				article.Score,
				article.Description,
				keywordTags)
		}

		html += `
            </div>
        </div>`
	}

	html += `
        <div class="footer">
            <p>📊 Generated by Advanced Tech News Collector</p>
            <p>🕒 Report generated at ` + time.Now().Format("15:04:05 MST") + `</p>
        </div>
    </div>
</body>
</html>`

	return html
}

// formatTimeAgo returns human-readable time difference
func formatTimeAgo(publishedAt time.Time) string {
	diff := time.Since(publishedAt)

	if diff < time.Minute {
		return "just now"
	} else if diff < time.Hour {
		minutes := int(diff.Minutes())
		return fmt.Sprintf("%d min ago", minutes)
	} else if diff < 24*time.Hour {
		hours := int(diff.Hours())
		return fmt.Sprintf("%d hours ago", hours)
	} else {
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%d days ago", days)
	}
}

// GenerateMarkdownReport creates a markdown report
func (nr *NewsReporter) GenerateMarkdownReport() string {
	var report strings.Builder

	report.WriteString(fmt.Sprintf("# 🚀 Tech News Report - %s\n\n", time.Now().Format("January 2, 2006")))
	report.WriteString(fmt.Sprintf("**Generated**: %s  \n", time.Now().Format("15:04:05 MST")))
	report.WriteString(fmt.Sprintf("**Total Articles**: %d  \n\n", len(nr.Articles)))

	// Group by category
	categorizedArticles := make(map[string][]NewsArticle)
	for _, article := range nr.Articles {
		categorizedArticles[article.Category] = append(categorizedArticles[article.Category], article)
	}

	categoryEmojis := map[string]string{
		"ai":      "🤖",
		"tech":    "💻",
		"digital": "💰",
		"hacking": "🔒",
	}

	for category, articles := range categorizedArticles {
		if len(articles) == 0 {
			continue
		}

		emoji := categoryEmojis[category]
		report.WriteString(fmt.Sprintf("## %s %s (%d articles)\n\n", emoji, strings.Title(category), len(articles)))

		for i, article := range articles {
			timeAgo := formatTimeAgo(article.PublishedAt)
			report.WriteString(fmt.Sprintf("### %d. [%s](%s)\n\n", i+1, article.Title, article.URL))
			report.WriteString(fmt.Sprintf("**Source**: %s | **Score**: 🔥 %d | **Published**: %s\n\n",
				article.Source, article.Score, timeAgo))

			if article.Description != "" {
				report.WriteString(fmt.Sprintf("%s\n\n", article.Description))
			}

			if len(article.Keywords) > 0 {
				report.WriteString("**Keywords**: ")
				for j, keyword := range article.Keywords {
					if j > 0 {
						report.WriteString(", ")
					}
					report.WriteString(fmt.Sprintf("`%s`", keyword))
				}
				report.WriteString("\n\n")
			}

			report.WriteString("---\n\n")
		}
	}

	report.WriteString("*Report generated by Advanced Tech News Collector*\n")
	return report.String()
}

// SaveReportToFile saves the report to a file
func (nr *NewsReporter) SaveReportToFile(format, filename string) error {
	var content string

	switch strings.ToLower(format) {
	case "html":
		content = nr.GenerateHTMLReport()
	case "markdown", "md":
		content = nr.GenerateMarkdownReport()
	case "json":
		jsonData, err := json.MarshalIndent(nr.Articles, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling JSON: %w", err)
		}
		content = string(jsonData)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	return os.WriteFile(filename, []byte(content), 0644)
}

// main function - entry point
func main() {
	// Get NewsAPI key from environment variable
	newsAPIKey := os.Getenv("NEWS_API_KEY")
	if newsAPIKey == "" {
		log.Println("Warning: NEWS_API_KEY not set. NewsAPI features will be disabled.")
	}

	// Create news collector
	collector := NewNewsCollector(newsAPIKey)

	fmt.Println("🚀 Starting Advanced Tech News Collection...")
	fmt.Println("📊 Collecting from multiple sources...")

	// Collect all news
	articles, err := collector.CollectAllNews()
	if err != nil {
		log.Fatalf("Error collecting news: %v", err)
	}

	if len(articles) == 0 {
		fmt.Println("❌ No articles found. Check your API keys and internet connection.")
		return
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Printf("📰 Found %d articles total\n", len(articles))

	// Create hot news detector
	hotDetector := NewImprovedHotNewsDetector()

	// Get hottest news
	hotArticles := hotDetector.GetHottestNews(articles, 20)
	fmt.Printf("🔥 %d hot articles detected\n", len(hotArticles))

	// Generate Twitter posts
	twitterPosts := generateEnhancedTwitterPosts(hotArticles)
	fmt.Printf("🐦 Generated %d Twitter posts\n", len(twitterPosts))

	// Create news reporter
	reporter := NewNewsReporter(hotArticles)

	// Save reports in different formats
	timestamp := time.Now().Format("2006-01-02_15-04-05")

	// Save HTML report
	htmlFile := fmt.Sprintf("tech_news_report_%s.html", timestamp)
	if err := reporter.SaveReportToFile("html", htmlFile); err != nil {
		log.Printf("Error saving HTML report: %v", err)
	} else {
		fmt.Printf("📄 HTML report saved: %s\n", htmlFile)
	}

	// Save Markdown report
	mdFile := fmt.Sprintf("tech_news_report_%s.md", timestamp)
	if err := reporter.SaveReportToFile("markdown", mdFile); err != nil {
		log.Printf("Error saving Markdown report: %v", err)
	} else {
		fmt.Printf("📝 Markdown report saved: %s\n", mdFile)
	}

	// Save JSON data
	jsonFile := fmt.Sprintf("tech_news_data_%s.json", timestamp)
	if err := reporter.SaveReportToFile("json", jsonFile); err != nil {
		log.Printf("Error saving JSON data: %v", err)
	} else {
		fmt.Printf("📊 JSON data saved: %s\n", jsonFile)
	}

	// Save Twitter posts
	twitterFile := fmt.Sprintf("twitter_posts_%s.json", timestamp)
	twitterData, _ := json.MarshalIndent(twitterPosts, "", "  ")
	if err := os.WriteFile(twitterFile, twitterData, 0644); err != nil {
		log.Printf("Error saving Twitter posts: %v", err)
	} else {
		fmt.Printf("🐦 Twitter posts saved: %s\n", twitterFile)
	}

	// Display summary
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("📊 COLLECTION SUMMARY")
	fmt.Println(strings.Repeat("=", 60))

	categoryStats := make(map[string]int)
	for _, article := range hotArticles {
		categoryStats[article.Category]++
	}

	for category, count := range categoryStats {
		emoji := map[string]string{
			"ai": "🤖", "tech": "💻", "digital": "💰", "hacking": "🔒",
		}[category]
		fmt.Printf("%s %s: %d articles\n", emoji, strings.Title(category), count)
	}

	fmt.Printf("\n🏆 Top Article: %s (Score: %d)\n", hotArticles[0].Title, hotArticles[0].Score)
	fmt.Printf("⏰ Collection completed in: %s\n", time.Since(time.Now()).Abs())
	fmt.Println("\n✅ All reports generated successfully!")
}
