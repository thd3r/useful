package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"
)

// NewsArticle represents a single news article
type NewsArticle struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	Source      string    `json:"source"`
	PublishedAt time.Time `json:"published_at"`
	Category    string    `json:"category"`
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

// NewsCollector manages news collection from multiple sources
type NewsCollector struct {
	NewsAPIKey string
	HTTPClient *http.Client
}

// NewNewsCollector creates a new news collector
func NewNewsCollector(apiKey string) *NewsCollector {
	return &NewsCollector{
		NewsAPIKey: apiKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetNewsAPIArticles fetches articles from NewsAPI
func (nc *NewsCollector) GetNewsAPIArticles(category string) ([]NewsArticle, error) {
	baseURL := "https://newsapi.org/v2/everything"

	// Define search queries for different categories
	queries := map[string]string{
		"ai":      "artificial intelligence OR machine learning OR deep learning OR AI OR neural networks",
		"tech":    "technology OR startup OR software OR hardware OR programming OR developer",
		"digital": "digital transformation OR cybersecurity OR blockchain OR cryptocurrency OR fintech",
		"hacking": "cybersecurity OR hacking OR data breach OR vulnerability OR malware OR ransomware",
	}

	query, exists := queries[category]
	if !exists {
		return nil, fmt.Errorf("unknown category: %s", category)
	}

	// Build URL with parameters
	params := url.Values{}
	params.Add("q", query)
	params.Add("language", "en")
	params.Add("sortBy", "publishedAt")
	params.Add("pageSize", "20")
	params.Add("from", time.Now().AddDate(0, 0, -1).Format("2006-01-02")) // Last 24 hours

	reqURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("X-API-Key", nc.NewsAPIKey) // Fixed: NewsAPI uses X-API-Key header
	req.Header.Set("User-Agent", "GoNewsCollector/1.0")

	resp, err := nc.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var newsResp NewsAPIResponse
	if err := json.Unmarshal(body, &newsResp); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}

	var articles []NewsArticle
	for _, article := range newsResp.Articles {
		// Parse published date
		publishedAt, err := time.Parse("2006-01-02T15:04:05Z", article.PublishedAt)
		if err != nil {
			// Try alternative format
			publishedAt, err = time.Parse("2006-01-02T15:04:05.000Z", article.PublishedAt)
			if err != nil {
				log.Printf("Error parsing date %s: %v", article.PublishedAt, err)
				publishedAt = time.Now()
			}
		}

		newsArticle := NewsArticle{
			Title:       article.Title,
			Description: article.Description,
			URL:         article.URL,
			Source:      article.Source.Name,
			PublishedAt: publishedAt,
			Category:    category,
		}
		articles = append(articles, newsArticle)
	}

	return articles, nil
}

// GetHackerNewsStories fetches top stories from Hacker News
func (nc *NewsCollector) GetHackerNewsStories() ([]NewsArticle, error) {
	// Get top story IDs
	topStoriesURL := "https://hacker-news.firebaseio.com/v0/topstories.json"
	resp, err := nc.HTTPClient.Get(topStoriesURL)
	if err != nil {
		return nil, fmt.Errorf("fetching top stories: %w", err)
	}
	defer resp.Body.Close()

	var storyIDs []int
	if err := json.NewDecoder(resp.Body).Decode(&storyIDs); err != nil {
		return nil, fmt.Errorf("decoding story IDs: %w", err)
	}

	var articles []NewsArticle

	// Fetch details for top 20 stories
	for i, id := range storyIDs {
		if i >= 20 {
			break
		}

		itemURL := fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", id)
		resp, err := nc.HTTPClient.Get(itemURL)
		if err != nil {
			log.Printf("Error fetching story %d: %v", id, err)
			continue
		}

		var item HackerNewsItem
		if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
			log.Printf("Error decoding story %d: %v", id, err)
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		// Filter for tech-related stories and skip empty titles
		if item.Title != "" && nc.isTechRelated(item.Title) {
			article := NewsArticle{
				Title:       item.Title,
				URL:         item.URL,
				Source:      "Hacker News",
				PublishedAt: time.Unix(item.Time, 0),
				Category:    "tech",
				Description: fmt.Sprintf("Score: %d, Comments: %d", item.Score, item.Descendants),
			}

			if article.URL == "" {
				article.URL = fmt.Sprintf("https://news.ycombinator.com/item?id=%d", item.ID)
			}

			articles = append(articles, article)
		}
	}

	return articles, nil
}

// isTechRelated checks if a title is tech-related
func (nc *NewsCollector) isTechRelated(title string) bool {
	techKeywords := []string{
		"AI", "artificial intelligence", "machine learning", "blockchain", "cryptocurrency",
		"software", "programming", "developer", "startup", "tech", "technology",
		"cybersecurity", "hacking", "data", "algorithm", "API", "cloud", "digital",
		"app", "mobile", "web", "internet", "computer", "system", "network",
	}

	titleLower := strings.ToLower(title)
	for _, keyword := range techKeywords {
		if strings.Contains(titleLower, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

// GetRedditTechNews fetches tech news from Reddit
func (nc *NewsCollector) GetRedditTechNews() ([]NewsArticle, error) {
	subreddits := []string{"technology", "artificial", "cybersecurity", "programming", "startups"}
	var allArticles []NewsArticle

	for _, subreddit := range subreddits {
		url := fmt.Sprintf("https://www.reddit.com/r/%s/hot.json?limit=10", subreddit)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Printf("Error creating Reddit request for %s: %v", subreddit, err)
			continue
		}

		req.Header.Set("User-Agent", "GoNewsCollector/1.0")

		resp, err := nc.HTTPClient.Do(req)
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
			// Skip empty titles and self posts without external URLs
			if child.Data.Title == "" || !strings.HasPrefix(child.Data.URL, "http") {
				continue
			}

			article := NewsArticle{
				Title:       child.Data.Title,
				URL:         child.Data.URL,
				Source:      fmt.Sprintf("Reddit r/%s", subreddit),
				PublishedAt: time.Unix(int64(child.Data.CreatedUTC), 0),
				Category:    "tech",
				Description: fmt.Sprintf("Score: %d", child.Data.Score),
			}

			allArticles = append(allArticles, article)
		}
	}

	return allArticles, nil
}

// CollectAllNews collects news from all sources
func (nc *NewsCollector) CollectAllNews() ([]NewsArticle, error) {
	var allArticles []NewsArticle

	categories := []string{"ai", "tech", "digital", "hacking"}

	// Collect from NewsAPI if API key is provided
	if nc.NewsAPIKey != "" {
		for _, category := range categories {
			articles, err := nc.GetNewsAPIArticles(category)
			if err != nil {
				log.Printf("Error fetching NewsAPI %s: %v", category, err)
				continue
			}
			allArticles = append(allArticles, articles...)
		}
	}

	// Collect from Hacker News
	hnArticles, err := nc.GetHackerNewsStories()
	if err != nil {
		log.Printf("Error fetching Hacker News: %v", err)
	} else {
		allArticles = append(allArticles, hnArticles...)
	}

	// Collect from Reddit
	redditArticles, err := nc.GetRedditTechNews()
	if err != nil {
		log.Printf("Error fetching Reddit: %v", err)
	} else {
		allArticles = append(allArticles, redditArticles...)
	}

	return allArticles, nil
}

// SaveToJSON saves articles to a JSON file
func (nc *NewsCollector) SaveToJSON(articles []NewsArticle, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	return encoder.Encode(articles)
}

// PrintSummary prints a summary of collected articles
func (nc *NewsCollector) PrintSummary(articles []NewsArticle) {
	fmt.Printf("\n=== NEWS SUMMARY FOR %s ===\n", time.Now().Format("2006-01-02"))
	fmt.Printf("Total articles collected: %d\n\n", len(articles))

	// Group by category
	categoryCount := make(map[string]int)
	sourceCount := make(map[string]int)

	for _, article := range articles {
		categoryCount[article.Category]++
		sourceCount[article.Source]++
	}

	fmt.Println("By Category:")
	for category, count := range categoryCount {
		fmt.Printf("  %s: %d articles\n", strings.Title(category), count)
	}

	fmt.Println("\nBy Source:")
	for source, count := range sourceCount {
		fmt.Printf("  %s: %d articles\n", source, count)
	}

	fmt.Println("\n=== LATEST ARTICLES ===")

	// Show latest 10 articles
	count := 0
	for _, article := range articles {
		if count >= 10 {
			break
		}

		fmt.Printf("\n[%s] %s\n", strings.ToUpper(article.Category), article.Title)
		fmt.Printf("Source: %s | Published: %s\n", article.Source, article.PublishedAt.Format("2006-01-02 15:04"))
		if article.Description != "" {
			fmt.Printf("Description: %s\n", article.Description)
		}
		fmt.Printf("URL: %s\n", article.URL)
		fmt.Println(strings.Repeat("-", 80))

		count++
	}
}

// HotNewsDetector analyzes and scores articles for hotness
type HotNewsDetector struct {
	HotKeywords    []string
	TrendingTopics []string
}

// NewHotNewsDetector creates a new hot news detector
func NewHotNewsDetector() *HotNewsDetector {
	return &HotNewsDetector{
		HotKeywords: []string{
			"breaking", "urgent", "alert", "massive", "billion", "breach", "hack",
			"fails", "crashes", "discontinue", "shutdown", "investment", "funding",
			"revolutionary", "breakthrough", "first-ever", "record", "largest",
			"attack", "vulnerability", "exploit", "leaked", "exposed", "stolen",
		},
		TrendingTopics: []string{
			"AI", "artificial intelligence", "ChatGPT", "OpenAI", "Google", "Microsoft",
			"cybersecurity", "data breach", "ransomware", "cryptocurrency", "blockchain",
			"startup", "IPO", "acquisition", "merger", "layoffs", "hiring",
		},
	}
}

// CalculateHotScore calculates hotness score for an article
func (hnd *HotNewsDetector) CalculateHotScore(article NewsArticle) int {
	score := 0
	titleLower := strings.ToLower(article.Title)
	descLower := strings.ToLower(article.Description)

	// Hot keywords bonus
	for _, keyword := range hnd.HotKeywords {
		if strings.Contains(titleLower, keyword) {
			score += 15
		}
		if strings.Contains(descLower, keyword) {
			score += 5
		}
	}

	// Trending topics bonus
	for _, topic := range hnd.TrendingTopics {
		if strings.Contains(titleLower, strings.ToLower(topic)) {
			score += 10
		}
	}

	// Recency bonus (newer = hotter)
	hoursOld := time.Since(article.PublishedAt).Hours()
	if hoursOld < 2 {
		score += 20
	} else if hoursOld < 6 {
		score += 15
	} else if hoursOld < 12 {
		score += 10
	} else if hoursOld < 24 {
		score += 5
	}

	// Source credibility bonus
	credibleSources := []string{"Reuters", "TechCrunch", "Wired", "Ars Technica", "The Verge"}
	for _, source := range credibleSources {
		if strings.Contains(article.Source, source) {
			score += 10
			break
		}
	}

	// Length penalty for very short titles
	if len(article.Title) < 30 {
		score -= 5
	}

	return score
}

// ScoredArticle represents an article with its hotness score
type ScoredArticle struct {
	Article NewsArticle
	Score   int
}

// GetHottestNews returns the hottest news articles
func (hnd *HotNewsDetector) GetHottestNews(articles []NewsArticle, limit int) []NewsArticle {
	var scored []ScoredArticle

	for _, article := range articles {
		score := hnd.CalculateHotScore(article)
		if score > 20 { // Minimum threshold for "hot" news
			scored = append(scored, ScoredArticle{
				Article: article,
				Score:   score,
			})
		}
	}

	// Sort by score (highest first) using sort package
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	// Return top articles
	var hotArticles []NewsArticle
	for i, scoredArticle := range scored {
		if i >= limit {
			break
		}
		hotArticles = append(hotArticles, scoredArticle.Article)
	}

	return hotArticles
}

// TwitterPost represents a Twitter post
type TwitterPost struct {
	Content  string `json:"content"`
	Source   string `json:"source"`
	URL      string `json:"url"`
	Snippet  string `json:"snippet"`
	Category string `json:"category"`
	Hashtags string `json:"hashtags"`
}

// TwitterPostGenerator generates Twitter posts from hot news
type TwitterPostGenerator struct {
	Templates map[string][]string
}

// NewTwitterPostGenerator creates a new Twitter post generator
func NewTwitterPostGenerator() *TwitterPostGenerator {
	return &TwitterPostGenerator{
		Templates: map[string][]string{
			"ai": {
				"🚨 BREAKING: %s\n\nAI world is shaking! %s\n\nSource: %s\n\n%s",
				"🤖 AI ALERT: %s\n\nThe future is changing fast! %s\n\nRead more: %s\n\n%s",
				"⚡ HOT: %s\n\nAI revolution continues! %s\n\nSource: %s\n\n%s",
			},
			"tech": {
				"🔥 TRENDING: %s\n\nTech world going crazy! %s\n\nSource: %s\n\n%s",
				"💥 BOOM: %s\n\nThis changes everything! %s\n\nRead: %s\n\n%s",
				"🚀 HUGE: %s\n\nTech scene is HOT! %s\n\nSource: %s\n\n%s",
			},
			"hacking": {
				"🚨 CYBER ALERT: %s\n\nStay safe out there! %s\n\nSource: %s\n\n%s",
				"⚠️ SECURITY BREACH: %s\n\nCheck your accounts NOW! %s\n\nRead: %s\n\n%s",
				"🔴 URGENT: %s\n\nCybersecurity under attack! %s\n\nSource: %s\n\n%s",
			},
			"digital": {
				"💸 MONEY MOVES: %s\n\nDigital economy shifting! %s\n\nSource: %s\n\n%s",
				"📈 TRENDING: %s\n\nFintech world heating up! %s\n\nRead: %s\n\n%s",
				"💰 BIG MOVES: %s\n\nDigital transformation! %s\n\nSource: %s\n\n%s",
			},
		},
	}
}

// GenerateTwitterPost creates a Twitter post from a news article
func (tpg *TwitterPostGenerator) GenerateTwitterPost(article NewsArticle) TwitterPost {
	templates, exists := tpg.Templates[article.Category]
	if !exists {
		templates = tpg.Templates["tech"] // Default to tech
	}

	// Select template based on current time for consistency during single run
	templateIndex := int(time.Now().Unix()) % len(templates)
	template := templates[templateIndex]

	// Truncate title if too long
	title := article.Title
	if len(title) > 80 {
		title = title[:77] + "..."
	}

	// Generate brief description
	brief := tpg.generateBrief(article)

	// Generate hashtags
	hashtags := tpg.generateHashtags(article)

	// Format post
	content := fmt.Sprintf(template, title, brief, article.Source, hashtags)

	// Ensure it's under Twitter limit (280 chars)
	if len(content) > 280 {
		// Truncate brief and try again
		brief = brief[:20] + "..."
		content = fmt.Sprintf(template, title, brief, article.Source, hashtags)

		if len(content) > 280 {
			content = content[:277] + "..."
		}
	}

	return TwitterPost{
		Content:  content,
		Source:   article.Source,
		URL:      article.URL,
		Snippet:  article.Description,
		Category: article.Category,
		Hashtags: hashtags,
	}
}

// generateBrief creates a brief description
func (tpg *TwitterPostGenerator) generateBrief(article NewsArticle) string {
	briefs := []string{
		"This is HUGE!",
		"Game changer!",
		"Mind = blown!",
		"The future is now!",
		"Incredible stuff!",
		"Buckle up!",
		"Plot twist!",
		"Next level!",
		"Revolutionary!",
		"Disruption mode!",
	}

	// Use article title hash for consistent brief selection
	index := len(article.Title) % len(briefs)
	return briefs[index]
}

// generateHashtags creates relevant hashtags
func (tpg *TwitterPostGenerator) generateHashtags(article NewsArticle) string {
	hashtagMap := map[string][]string{
		"ai":      {"#AI", "#MachineLearning", "#TechNews", "#Innovation", "#Future"},
		"tech":    {"#Tech", "#Innovation", "#Startup", "#TechNews", "#Digital"},
		"hacking": {"#Cybersecurity", "#InfoSec", "#Security", "#CyberAlert", "#Hacking"},
		"digital": {"#Fintech", "#Digital", "#Crypto", "#Blockchain", "#Investment"},
	}

	tags, exists := hashtagMap[article.Category]
	if !exists {
		tags = hashtagMap["tech"]
	}

	// Select 3-4 most relevant hashtags
	var selectedTags []string
	for i := 0; i < 4 && i < len(tags); i++ {
		selectedTags = append(selectedTags, tags[i])
	}

	return strings.Join(selectedTags, " ")
}

// saveTwitterPostsToJSON saves Twitter posts to JSON file
func saveTwitterPostsToJSON(posts []TwitterPost, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	return encoder.Encode(posts)
}

// GetMarkdownReport generates a markdown report
func (nc *NewsCollector) GetMarkdownReport(articles []NewsArticle) string {
	var md strings.Builder

	md.WriteString(fmt.Sprintf("# Tech News Report - %s\n\n", time.Now().Format("January 2, 2006")))

	categories := map[string][]NewsArticle{
		"AI & Machine Learning": {},
		"Technology":            {},
		"Digital & Fintech":     {},
		"Cybersecurity":         {},
	}

	for _, article := range articles {
		switch article.Category {
		case "ai":
			categories["AI & Machine Learning"] = append(categories["AI & Machine Learning"], article)
		case "tech":
			categories["Technology"] = append(categories["Technology"], article)
		case "digital":
			categories["Digital & Fintech"] = append(categories["Digital & Fintech"], article)
		case "hacking":
			categories["Cybersecurity"] = append(categories["Cybersecurity"], article)
		}
	}

	for category, arts := range categories {
		if len(arts) == 0 {
			continue
		}

		md.WriteString(fmt.Sprintf("## %s\n\n", category))

		for _, article := range arts {
			md.WriteString(fmt.Sprintf("### [%s](%s)\n", article.Title, article.URL))
			md.WriteString(fmt.Sprintf("**Source:** %s | **Published:** %s\n\n",
				article.Source, article.PublishedAt.Format("2006-01-02 15:04")))

			if article.Description != "" {
				md.WriteString(fmt.Sprintf("%s\n\n", article.Description))
			}

			md.WriteString("---\n\n")
		}
	}

	return md.String()
}

func main() {
	// Get NewsAPI key from environment variable
	newsAPIKey := os.Getenv("NEWS_API_KEY")
	if newsAPIKey == "" {
		fmt.Println("Warning: NEWS_API_KEY not set. NewsAPI features will be disabled.")
		fmt.Println("Get your free API key from: https://newsapi.org/")
	}

	// Create news collector
	collector := NewNewsCollector(newsAPIKey)

	fmt.Println("🔍 Collecting latest AI, Tech, Digital & Hacking news...")

	// Collect all news
	articles, err := collector.CollectAllNews()
	if err != nil {
		log.Fatalf("Error collecting news: %v", err)
	}

	if len(articles) == 0 {
		fmt.Println("No articles found. Please check your API key and internet connection.")
		return
	}

	// Detect hot news
	detector := NewHotNewsDetector()
	hotArticles := detector.GetHottestNews(articles, 10)

	fmt.Printf("\n🔥 Found %d HOT articles!\n", len(hotArticles))

	// Generate Twitter posts
	generator := NewTwitterPostGenerator()
	var twitterPosts []TwitterPost

	for _, article := range hotArticles {
		post := generator.GenerateTwitterPost(article)
		twitterPosts = append(twitterPosts, post)
	}

	// Print hot news and Twitter posts
	fmt.Println("\n=== 🔥 HOTTEST NEWS & TWITTER POSTS ===")

	for i, post := range twitterPosts {
		fmt.Printf("\n🔥 HOT NEWS #%d:\n", i+1)
		fmt.Printf("Category: %s\n", strings.ToUpper(post.Category))
		fmt.Printf("Source: %s\n", post.Source)
		fmt.Printf("Snippet: %s\n", post.Snippet)
		fmt.Printf("URL: %s\n", post.URL)
		fmt.Printf("\n📱 TWITTER POST:\n%s\n", post.Content)
		fmt.Println(strings.Repeat("=", 80))
	}

	// Save Twitter posts to JSON
	twitterFilename := fmt.Sprintf("hot_twitter_posts_%s.json", time.Now().Format("2006-01-02"))
	if err := saveTwitterPostsToJSON(twitterPosts, twitterFilename); err != nil {
		log.Printf("Error saving Twitter posts: %v", err)
	} else {
		fmt.Printf("\n✅ Twitter posts saved to: %s\n", twitterFilename)
	}

	// Save all articles to JSON
	allFilename := fmt.Sprintf("tech_news_%s.json", time.Now().Format("2006-01-02"))
	if err := collector.SaveToJSON(articles, allFilename); err != nil {
		log.Printf("Error saving all articles: %v", err)
	} else {
		fmt.Printf("✅ All articles saved to: %s\n", allFilename)
	}

	fmt.Println("\n🎉 Hot news detection and Twitter post generation completed!")
	fmt.Printf("📊 Summary: %d total articles → %d hot articles → %d Twitter posts\n",
		len(articles), len(hotArticles), len(twitterPosts))
}
