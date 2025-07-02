package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"html"
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
				"Gemini", "Google Gemini", "Bard", "Google Veo", "Veo 3", "Claude", "Anthropic",
				"Midjourney", "DALL-E", "Stable Diffusion", "Runway ML", "Sora", "Udio",
				"AI assistant", "AI chatbot", "AI art", "AI video", "AI music", "text-to-image",
				"text-to-video", "voice AI", "speech synthesis", "AI ethics", "AGI", "AI safety",
				"prompt engineering", "fine-tuning", "RAG", "vector database", "embeddings",
				"multimodal AI", "foundation model", "large language model", "neural architecture",
				"artificial intelligence", "machine learning", "deep learning", "neural network",
				"Microsoft Copilot", "GitHub Copilot", "Midjourney", "Stable Diffusion", "DALL-E",
				"Hugging Face", "Perplexity AI", "Llama", "Meta AI", "Google Bard", "AI agents",
				"AGI", "artificial general intelligence", "AI safety", "AI ethics", "AI regulation",
			},
			SecondaryKeywords: []string{
				"algorithm", "automation", "robot", "cognitive", "intelligent", "smart system",
				"predictive", "recommendation", "pattern recognition", "data mining",
				"AI-powered", "AI-driven", "machine intelligence", "computational intelligence",
				"artificial neural", "backpropagation", "gradient descent", "attention mechanism",
				"decoder", "encoder", "tokenizer", "inference", "training data", "model weights",
				"neural processing", "edge AI", "AI accelerator", "MLOps", "AutoML",
			},
			ExcludeKeywords: []string{
				"game", "movie", "entertainment", "sport", "weather", "fashion", "celebrity",
			},
			MinScore:         25,
			WeightMultiplier: 1.5,
		},
		"tech": {
			PrimaryKeywords: []string{
				"startup", "technology", "software", "hardware", "programming", "developer",
				"innovation", "digital transformation", "cloud computing", "SaaS", "API",
				"mobile app", "web development", "IoT", "5G", "quantum computing",
				"venture capital", "VC funding", "Series A", "Series B", "IPO", "acquisition",
				"unicorn startup", "decacorn", "Y Combinator", "Techstars", "accelerator",
				"React", "Vue", "Angular", "Node.js", "Python", "JavaScript", "TypeScript",
				"Kubernetes", "Docker", "microservices", "serverless", "edge computing",
				"DevOps", "CI/CD", "GitHub", "GitLab", "AWS", "Azure", "Google Cloud",
				"database", "MongoDB", "PostgreSQL", "Redis", "Elasticsearch", "GraphQL",
				"cybersecurity", "data science", "analytics", "big data", "data engineering",
				"product management", "agile", "scrum", "tech layoffs", "remote work",
			},
			SecondaryKeywords: []string{
				"tech company", "silicon valley", "venture capital", "funding", "IPO",
				"platform", "framework", "open source", "github", "coding",
				"software engineer", "full stack", "frontend", "backend", "mobile developer",
				"UI/UX", "product design", "tech stack", "architecture", "scalability",
				"performance", "optimization", "debugging", "testing", "deployment",
				"code review", "technical debt", "refactoring", "legacy system",
			},
			ExcludeKeywords: []string{
				"celebrity", "politics", "entertainment", "sport", "weather", "fashion",
			},
			MinScore:         20,
			WeightMultiplier: 1.2,
		},
		"digital": {
			PrimaryKeywords: []string{
				"cryptocurrency", "blockchain", "bitcoin", "ethereum", "fintech", "digital payment",
				"NFT", "DeFi", "digital currency", "crypto", "digital wallet", "trading",
				"financial technology", "digital banking", "e-commerce", "digital economy",
				"Web3", "decentralized", "smart contract", "dApp", "DAO", "staking",
				"yield farming", "liquidity mining", "AMM", "DEX", "CEX", "crypto exchange",
				"altcoin", "stablecoin", "CBDC", "digital asset", "tokenization", "minting",
				"metaverse", "virtual reality", "augmented reality", "VR", "AR", "XR",
				"digital twin", "IoT", "connected devices", "smart city", "Industry 4.0",
				"PayPal", "Stripe", "Square", "Revolut", "Robinhood", "Coinbase", "Binance",
				"mobile payment", "contactless payment", "QR payment", "BNPL", "neobank",
				"regtech", "insurtech", "wealthtech", "proptech", "lending platform",
				"robo-advisor", "algorithmic trading", "high-frequency trading", "arbitrage",
				"blockchain", "distributed ledger", "consensus mechanism", "proof of stake",
				"proof of work", "layer 2", "scaling solution", "interoperability",
				"cross chain", "bridge protocol", "rollups", "sidechains", "sharding",
				"validator", "mining", "staking", "governance token", "tokenomics",
				"blockchain development", "solidity", "smart contract audit", "oracles",
			},
			SecondaryKeywords: []string{
				"investment", "market", "finance", "trading platform", "exchange",
				"digital transformation", "online payment", "mobile payment",
				"financial services", "banking", "credit", "lending", "insurance",
				"wealth management", "portfolio", "asset management", "risk management",
				"compliance", "KYC", "AML", "regulatory", "SEC", "CFTC", "regulation",
				"market cap", "volume", "volatility", "bull market", "bear market",
				"decentralized", "trustless", "immutable", "transparent", "permissionless",
				"consensus", "node", "hash", "merkle tree", "cryptography",
			},
			ExcludeKeywords: []string{
				"celebrity", "entertainment", "sport", "weather", "politics unrelated", "fashion",
			},
			MinScore:         22,
			WeightMultiplier: 1.3,
		},
		"hacking": {
			PrimaryKeywords: []string{
				"cybersecurity", "data breach", "hacking", "cyber attack", "malware",
				"ransomware", "phishing", "vulnerability", "security breach", "cyber threat",
				"information security", "network security", "data leak", "exploit",
				"zero-day", "APT", "advanced persistent threat", "SQL injection", "XSS",
				"CSRF", "buffer overflow", "privilege escalation", "lateral movement",
				"social engineering", "spear phishing", "whaling", "business email compromise",
				"cryptojacking", "botnet", "trojan", "rootkit", "keylogger", "spyware",
				"DDoS", "distributed denial of service", "man-in-the-middle", "MITM",
				"endpoint security", "SIEM", "SOC", "incident response", "threat hunting",
				"penetration testing", "red team", "blue team", "purple team", "bug bounty",
				"CVE", "patch management", "vulnerability assessment", "security audit",
				"GDPR", "CCPA", "privacy", "data protection", "encryption", "decryption",
				"PKI", "certificate", "digital signature", "two-factor authentication", "2FA",
				"multi-factor authentication", "MFA", "biometric", "password manager",
			},
			SecondaryKeywords: []string{
				"security", "privacy", "encryption", "firewall", "antivirus",
				"penetration testing", "ethical hacking", "cyber defense",
				"security researcher", "white hat", "black hat", "grey hat", "hacktivist",
				"cyber warfare", "nation-state", "threat actor", "attack vector",
				"security framework", "NIST", "ISO 27001", "compliance", "audit",
				"risk assessment", "security awareness", "training", "phishing simulation",
				"security tools", "scanner", "forensics", "malware analysis",
			},
			ExcludeKeywords: []string{
				"movie", "game", "entertainment", "sport", "weather", "fashion", "celebrity",
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
			Timeout: 15 * time.Second,
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
		// AI keywords
		"ai", "artificial intelligence", "machine learning", "chatgpt", "openai",
		"gemini", "claude", "midjourney", "dall-e", "stable diffusion", "neural network",
		"deep learning", "llm", "generative ai", "computer vision", "nlp",
		"ai", "artificial intelligence", "machine learning", "deep learning",
		"chatgpt", "openai", "google gemini", "gemini ai", "google veo", "veo ai",
		"claude ai", "anthropic", "microsoft copilot", "github copilot", "google bard",
		"perplexity ai", "midjourney", "stable diffusion", "dall-e", "hugging face",
		"meta ai", "llama", "gpt-4", "gpt-5", "neural network", "transformer",
		"computer vision", "nlp", "reinforcement learning", "agi", "ai agents",

		// Blockchain & Crypto
		"blockchain", "cryptocurrency", "bitcoin", "ethereum", "solana", "cardano",
		"polygon", "avalanche", "binance", "coinbase", "defi", "nft", "web3", "dao",
		"smart contracts", "stablecoin", "cbdc", "gamefi", "metaverse",

		// Cybersecurity
		"cybersecurity", "data breach", "ransomware", "malware", "phishing",
		"zero day", "apt", "supply chain attack", "endpoint security",
		"cloud security", "iot security", "incident response", "threat intelligence",

		// Tech Companies
		"microsoft", "apple", "google", "meta", "amazon", "tesla", "nvidia",
		"amd", "intel", "qualcomm", "samsung", "tsmc", "broadcom", "AT&T", "IBM",

		// Fintech
		"fintech", "digital payment", "neo banking", "bnpl", "robo advisor",
		"insurtech", "regtech", "wealthtech", "proptech", "paytech",

		// Tech keywords
		"startup", "funding", "ipo", "acquisition", "venture capital", "silicon valley",
		"programming", "developer", "software", "hardware", "cloud", "saas", "api",
		"react", "javascript", "python", "kubernetes", "docker", "aws", "github",
		"cybersecurity", "data science", "mobile app", "web development", "devops",

		// Digital keywords
		"cryptocurrency", "bitcoin", "ethereum", "blockchain", "nft", "defi", "web3",
		"fintech", "digital payment", "crypto", "stablecoin", "metaverse", "dao",
		"smart contract", "trading", "exchange", "wallet", "mining", "staking",

		// Hacking keywords
		"cybersecurity", "data breach", "ransomware", "malware", "phishing", "exploit",
		"vulnerability", "zero-day", "apt", "ddos", "penetration testing", "bug bounty",
		"encryption", "firewall", "antivirus", "threat", "attack", "security",
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
		"digital": "\"cryptocurrency\" OR \"blockchain\" OR \"bitcoin\" OR \"fintech\" OR \"digital payment\" OR \"NFT\" OR \"DeFi\"",
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
	params.Add("pageSize", "100")                                         // Increased to get more candidates
	params.Add("from", time.Now().AddDate(0, 0, -3).Format("2006-01-02")) // Last 3 days

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
	subreddits := []string{"technology", "artificial", "cybersecurity", "programming", "blockchain", "MachineLearning"}
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
			"shocking", "stunning", "dramatic", "historic", "landmark", "game-changing",
			"disruptive", "groundbreaking", "cutting-edge", "next-generation",
		},
		TrendingTopics: []string{
			// AI trending topics
			"ChatGPT", "GPT-4", "GPT-5", "OpenAI", "Google Gemini", "Gemini Pro", "Claude", "Anthropic",
			"Google Veo", "Veo 3", "Sora", "Midjourney", "DALL-E 3", "Stable Diffusion", "Runway",
			"AI safety", "AGI", "artificial general intelligence", "AI regulation", "AI ethics",
			"ChatGPT", "OpenAI", "Google Gemini", "Gemini AI", "Google Veo", "Veo AI",
			"Claude AI", "Anthropic", "Microsoft Copilot", "GitHub Copilot", "Google Bard",
			"Perplexity AI", "Midjourney", "Stable Diffusion", "DALL-E", "Hugging Face",
			"Meta AI", "Llama", "GPT-4", "GPT-5", "AGI", "AI agents",

			// Tech Giants
			"Microsoft", "Apple", "Google", "Meta", "Amazon", "Tesla", "NVIDIA",
			"AMD", "Intel", "Qualcomm", "Samsung", "TSMC", "Broadcom", "IBM", "AT&T",

			// Crypto & Blockchain
			"Bitcoin", "Ethereum", "Solana", "Cardano", "Polygon", "Avalanche",
			"Chainlink", "Uniswap", "Binance", "Coinbase", "FTX", "Celsius",
			"Terra Luna", "stablecoin", "USDC", "Tether", "DeFi", "NFT",
			"Web3", "DAO", "GameFi", "metaverse",

			// Cybersecurity
			"cybersecurity", "data breach", "ransomware", "zero day", "APT",
			"supply chain attack", "SolarWinds", "Log4j", "critical vulnerability",

			// Tech trending topics
			"Apple Vision Pro", "Microsoft", "Apple", "Tesla", "Meta", "Amazon", "Netflix", "Spotify",
			"TikTok ban", "Twitter X", "Instagram", "YouTube", "LinkedIn", "GitHub Copilot",
			"startup funding", "IPO", "acquisition", "layoffs", "hiring freeze", "remote work",
			"venture capital", "unicorn", "Y Combinator", "Techstars", "Silicon Valley",

			// Digital trending topics
			"Bitcoin ETF", "Ethereum 2.0", "Solana", "Cardano", "Polygon", "Avalanche", "Chainlink",
			"crypto regulation", "SEC", "cryptocurrency crash", "bull run", "altcoin season",
			"NFT marketplace", "OpenSea", "Blur", "metaverse", "Web3", "DeFi protocol",
			"stablecoin", "USDC", "USDT", "Binance", "Coinbase", "FTX", "crypto exchange",

			// Hacking trending topics
			"data breach", "ransomware attack", "zero-day exploit", "cyber attack", "phishing campaign",
			"malware outbreak", "security vulnerability", "patch Tuesday", "bug bounty", "CVE",
			"nation-state hacking", "APT group", "cybersecurity", "GDPR fine", "privacy violation",
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
			"🤖 AI BREAKING: %s\n\nThe future is NOW! This changes everything in tech.\n\n %s\n\n🔗 %s\n\n%s",
			"⚡ MASSIVE AI NEWS: %s\n\nMind = BLOWN 🤯 The AI revolution accelerates!\n\n%s\n\n🔗 %s\n\n%s",
			"🚨 AI ALERT: %s\n\nThis is HUGE for artificial intelligence! \n\n%s\n\n🔗 %s\n\n%s",
		},
		"tech": {
			"🔥 TECH BOMB: %s\n\nSilicon Valley is BUZZING! This is massive! \n\n%s\n\n🔗 %s\n\n%s",
			"🚀 STARTUP ALERT: %s\n\nThe tech world just shifted! Game changer! \n\n%s\n\n🔗 %s\n\n%s",
			"💡 INNOVATION BLAST: %s\n\nTech disruption in real-time! Don't miss this! \n\n%s\n\n🔗 %s\n\n%s",
		},
		"hacking": {
			"🚨 CYBER CRISIS: %s\n\nMajor security alert! Check your systems NOW! \n\n%s\n\n🔗 %s\n\n%s",
			"⚠️ BREACH ALERT: %s\n\nCybersecurity nightmare unfolding! Stay protected! \n\n%s\n\n🔗 %s\n\n%s",
			"🔴 SECURITY EMERGENCY: %s\n\nHackers are evolving! Critical update needed! \n\n%s\n\n🔗 %s\n\n%s",
		},
		"digital": {
			"💰 CRYPTO EXPLOSION: %s\n\nThe digital economy is on fire! \n\n%s\n\n🔗 %s\n\n%s",
			"🚀 BLOCKCHAIN BOOM: %s\n\nFintech revolution happening NOW! Don't miss out! \n\n%s\n\n🔗 %s\n\n%s",
			"💎 DIGITAL GOLD RUSH: %s\n\nThe future of money is here! Game changer! \n\n%s\n\n🔗 %s\n\n%s",
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
			article.Title,
			article.Description,
			article.URL,
			hashtags,
		)

		// Ensure post is within Twitter character limit
		// if len(post) > 280 {
		// 	post = truncatePost(post, 280)
		// }

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

// GenerateHTMLReport creates a beautiful HTML report with modern design and theme toggle
func (nr *NewsReporter) GenerateHTMLReport() string {
	htmlContent := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Tech News Report - ` + time.Now().Format("January 2, 2006") + `</title>
    <style>
        :root {
            --bg-primary: #0a0a0a;
            --bg-secondary: #1a1a1a;
            --bg-card: #2a2a2a;
            --bg-accent: #333333;
            --text-primary: #ffffff;
            --text-secondary: #b3b3b3;
            --text-muted: #808080;
            --border-color: #404040;
            --accent-ai: #ff6b6b;
            --accent-tech: #4ecdc4;
            --accent-digital: #45b7d1;
            --accent-hacking: #f9ca24;
            --gradient-primary: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            --gradient-card: linear-gradient(145deg, #2a2a2a 0%, #1f1f1f 100%);
            --shadow-light: 0 4px 20px rgba(255, 255, 255, 0.05);
            --shadow-heavy: 0 10px 40px rgba(0, 0, 0, 0.3);
        }

        [data-theme="light"] {
            --bg-primary: #ffffff;
            --bg-secondary: #f8f9fa;
            --bg-card: #ffffff;
            --bg-accent: #e9ecef;
            --text-primary: #212529;
            --text-secondary: #495057;
            --text-muted: #6c757d;
            --border-color: #dee2e6;
            --accent-ai: #dc3545;
            --accent-tech: #20c997;
            --accent-digital: #0d6efd;
            --accent-hacking: #ffc107;
            --gradient-primary: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            --gradient-card: linear-gradient(145deg, #ffffff 0%, #f8f9fa 100%);
            --shadow-light: 0 4px 20px rgba(0, 0, 0, 0.08);
            --shadow-heavy: 0 10px 40px rgba(0, 0, 0, 0.15);
        }

        * { 
            margin: 0; 
            padding: 0; 
            box-sizing: border-box; 
        }

        body { 
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            line-height: 1.6; 
            color: var(--text-primary); 
            background: var(--bg-primary);
            min-height: 100vh;
            scroll-behavior: smooth;
            transition: background-color 0.3s ease, color 0.3s ease;
        }

        .container { 
            max-width: 1400px; 
            margin: 0 auto; 
            padding: 20px;
        }

        .theme-toggle {
            position: fixed;
            top: 20px;
            right: 20px;
            z-index: 1000;
            background: var(--bg-card);
            border: 2px solid var(--border-color);
            border-radius: 50px;
            padding: 12px 16px;
            cursor: pointer;
            transition: all 0.3s ease;
            box-shadow: var(--shadow-light);
            display: flex;
            align-items: center;
            gap: 8px;
            font-size: 14px;
            font-weight: 600;
            color: var(--text-primary);
        }

        .theme-toggle:hover {
            transform: translateY(-2px);
            box-shadow: var(--shadow-heavy);
            border-color: var(--accent-tech);
        }

        .theme-icon {
            font-size: 16px;
            transition: transform 0.3s ease;
        }

        .theme-toggle:hover .theme-icon {
            transform: rotate(180deg);
        }

        .header {
            text-align: center;
            margin-bottom: 60px;
            padding: 60px 20px;
            background: var(--gradient-primary);
            border-radius: 24px;
            position: relative;
            overflow: hidden;
        }

        .header::before {
            content: '';
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: url('data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><defs><pattern id="grid" width="10" height="10" patternUnits="userSpaceOnUse"><path d="M 10 0 L 0 0 0 10" fill="none" stroke="rgba(255,255,255,0.1)" stroke-width="0.5"/></pattern></defs><rect width="100" height="100" fill="url(%23grid)"/></svg>');
            opacity: 0.3;
        }

        .header-content {
            position: relative;
            z-index: 1;
        }

        .header h1 {
            font-size: clamp(2.5rem, 5vw, 4rem);
            font-weight: 800;
            margin-bottom: 16px;
            background: linear-gradient(45deg, #ffffff, #e0e0e0);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
            text-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
        }

        .header-subtitle {
            font-size: 1.25rem;
            opacity: 0.9;
            font-weight: 300;
            letter-spacing: 0.5px;
        }

        .header-date {
            margin-top: 12px;
            font-size: 1rem;
            opacity: 0.8;
            font-weight: 500;
        }

        .stats {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
            gap: 24px;
            margin-bottom: 60px;
        }

        .stat-card {
            background: var(--gradient-card);
            padding: 32px 24px;
            border-radius: 20px;
            text-align: center;
            border: 1px solid var(--border-color);
            box-shadow: var(--shadow-light);
            transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
            position: relative;
            overflow: hidden;
        }

        .stat-card::before {
            content: '';
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            height: 3px;
            background: var(--gradient-primary);
            transform: scaleX(0);
            transition: transform 0.3s ease;
        }

        .stat-card:hover {
            transform: translateY(-4px);
            box-shadow: var(--shadow-heavy);
            border-color: rgba(255, 255, 255, 0.2);
        }

        [data-theme="light"] .stat-card:hover {
            border-color: rgba(0, 0, 0, 0.2);
        }

        .stat-card:hover::before {
            transform: scaleX(1);
        }

        .stat-number {
            font-size: 3rem;
            font-weight: 800;
            display: block;
            color: var(--text-primary);
            margin-bottom: 8px;
            background: var(--gradient-primary);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
        }

        .stat-label {
            font-size: 1rem;
            color: var(--text-secondary);
            font-weight: 500;
            text-transform: uppercase;
            letter-spacing: 1px;
        }

        .category-section {
            margin-bottom: 60px;
            background: var(--bg-secondary);
            border-radius: 24px;
            overflow: hidden;
            border: 1px solid var(--border-color);
            box-shadow: var(--shadow-light);
        }

        .category-header {
            padding: 32px;
            background: var(--gradient-card);
            border-bottom: 1px solid var(--border-color);
            position: relative;
            cursor: pointer;
            transition: all 0.3s ease;
            user-select: none;
        }

        .category-header:hover {
            background: var(--bg-card);
        }

        .category-header::before {
            content: '';
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            height: 4px;
        }

        .category-ai::before { background: var(--accent-ai); }
        .category-tech::before { background: var(--accent-tech); }
        .category-digital::before { background: var(--accent-digital); }
        .category-hacking::before { background: var(--accent-hacking); }
        
        .category-title {
            font-size: 1.75rem;
            font-weight: 700;
            color: var(--text-primary);
            display: flex;
            align-items: center;
            gap: 16px;
            width: 100%;
        }

        .category-toggle {
            margin-left: auto;
            font-size: 1.5rem;
            transition: transform 0.3s ease;
            color: var(--text-secondary);
        }

        .category-section.collapsed .category-toggle {
            transform: rotate(-90deg);
        }

        .category-icon {
            font-size: 2rem;
            opacity: 0.9;
        }

        .category-count {
            background: var(--bg-accent);
            padding: 8px 16px;
            border-radius: 20px;
            font-size: 0.9rem;
            font-weight: 600;
            color: var(--text-secondary);
        }

        .articles-grid {
            padding: 32px;
            display: grid;
            gap: 24px;
            transition: all 0.3s ease;
            overflow: hidden;
        }

        .category-section.collapsed .articles-grid {
            max-height: 0;
            padding: 0 32px;
            opacity: 0;
        }

        .category-section:not(.collapsed) .articles-grid {
            max-height: none;
            opacity: 1;
        }

        .article-card {
            background: var(--gradient-card);
            padding: 28px;
            border-radius: 16px;
            border: 1px solid var(--border-color);
            transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
            position: relative;
            overflow: hidden;
        }

        .article-card::before {
            content: '';
            position: absolute;
            top: 0;
            left: 0;
            width: 4px;
            height: 100%;
            transition: all 0.3s ease;
        }

        .article-ai::before { background: var(--accent-ai); }
        .article-tech::before { background: var(--accent-tech); }
        .article-digital::before { background: var(--accent-digital); }
        .article-hacking::before { background: var(--accent-hacking); }

        .article-card:hover {
            transform: translateY(-2px);
            box-shadow: var(--shadow-heavy);
            border-color: rgba(255, 255, 255, 0.15);
        }

        [data-theme="light"] .article-card:hover {
            border-color: rgba(0, 0, 0, 0.15);
        }

        .article-title {
            font-size: 1.4rem;
            font-weight: 700;
            margin-bottom: 16px;
            line-height: 1.4;
        }

        .article-title a {
            color: var(--text-primary);
            text-decoration: none;
            transition: color 0.2s ease;
        }

        .article-title a:hover {
            color: var(--accent-digital);
        }

        .article-meta {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 16px;
            flex-wrap: wrap;
            gap: 12px;
        }

        .article-source {
            color: var(--text-muted);
            font-size: 0.9rem;
            font-weight: 500;
        }

        .article-time {
            color: var(--text-muted);
            font-size: 0.85rem;
        }

        .hot-score {
            background: linear-gradient(45deg, var(--accent-ai), #ee5a6f);
            color: white;
            padding: 8px 16px;
            border-radius: 20px;
            font-weight: 700;
            font-size: 0.85rem;
            box-shadow: 0 4px 12px rgba(255, 107, 107, 0.3);
            white-space: nowrap;
        }

        .article-description {
            color: var(--text-secondary);
            margin-bottom: 20px;
            line-height: 1.6;
            font-size: 0.95rem;
        }

        .keywords {
            display: flex;
            flex-wrap: wrap;
            gap: 8px;
        }

        .keyword-tag {
            background: var(--bg-accent);
            color: var(--text-secondary);
            padding: 6px 12px;
            border-radius: 16px;
            font-size: 0.8rem;
            font-weight: 500;
            border: 1px solid var(--border-color);
            transition: all 0.2s ease;
        }

        .keyword-tag:hover {
            background: var(--border-color);
            color: var(--text-primary);
        }

        .footer {
            text-align: center;
            color: var(--text-muted);
            margin-top: 80px;
            padding: 40px 20px;
            background: var(--bg-secondary);
            border-radius: 24px;
            border: 1px solid var(--border-color);
        }

        .footer-logo {
            font-size: 1.2rem;
            font-weight: 700;
            margin-bottom: 12px;
            color: var(--text-primary);
        }

        .footer-time {
            font-size: 0.9rem;
            opacity: 0.8;
        }
		
		.footer-copyright {
			font-size: 0.9rem;
			opacity: 0.8;
		}

        /* Loading animation */
        @keyframes pulse {
            0%, 100% { opacity: 1; }
            50% { opacity: 0.5; }
        }

        .loading { animation: pulse 2s infinite; }

        /* Responsive Design */
        @media (max-width: 768px) {
            .container { padding: 16px; }
            .header { padding: 40px 20px; margin-bottom: 40px; }
            .stats { grid-template-columns: 1fr; gap: 16px; }
            .stat-card { padding: 24px 20px; }
            .category-header { padding: 24px 20px; }
            .articles-grid { padding: 20px; gap: 20px; }
            .article-card { padding: 20px; }
            .category-title { font-size: 1.5rem; flex-direction: column; align-items: flex-start; gap: 12px; }
            .category-count { margin-left: 0; }
            .category-toggle { display: none; }
            .article-meta { flex-direction: column; align-items: flex-start; }
            .theme-toggle { 
                position: fixed; 
                top: 10px; 
                right: 10px; 
                padding: 10px 12px; 
                font-size: 12px; 
            }
        }

        @media (max-width: 480px) {
            .article-title { font-size: 1.2rem; }
            .category-title { font-size: 1.3rem; }
            .stat-number { font-size: 2.5rem; }
            .theme-toggle { 
                padding: 8px 10px; 
                font-size: 11px; 
            }
        }

        /* Smooth scrolling for better UX */
        html { scroll-behavior: smooth; }

        /* Focus styles for accessibility */
        a:focus, button:focus, .theme-toggle:focus {
            outline: 2px solid var(--accent-tech);
            outline-offset: 2px;
            border-radius: 4px;
        }

        /* Print styles */
        @media print {
            .theme-toggle { display: none !important; }
            .header { background: none !important; color: black !important; }
            .stat-card, .category-section, .article-card { 
                break-inside: avoid; 
                box-shadow: none !important;
                border: 1px solid #ccc !important;
            }
        }
    </style>
</head>
<body>
    <button class="theme-toggle" onclick="toggleTheme()" title="Toggle theme">
        <span class="theme-icon">🌙</span>
        <span class="theme-text">Dark</span>
    </button>

    <div class="container">
        <header class="header">
            <div class="header-content">
                <h1>🚀 Tech News Report</h1>
                <p class="header-subtitle">Latest Technology News & Insights</p>
                <div class="header-date">` + time.Now().Format("Monday, January 2, 2006") + `</div>
            </div>
        </header>`

	// Calculate statistics
	categoryStats := make(map[string]int)
	totalScore := 0
	oldestTime := time.Now()

	for _, article := range nr.Articles {
		categoryStats[article.Category]++
		totalScore += article.Score
		if article.PublishedAt.Before(oldestTime) {
			oldestTime = article.PublishedAt
		}
	}

	avgScore := 0
	hoursAgo := 0.0
	if len(nr.Articles) > 0 {
		avgScore = totalScore / len(nr.Articles)
		hoursAgo = time.Since(oldestTime).Hours()
	}

	// Add statistics section
	htmlContent += fmt.Sprintf(`
        <section class="stats">
            <div class="stat-card">
                <span class="stat-number">%d</span>
                <div class="stat-label">Total Articles</div>
            </div>
            <div class="stat-card">
                <span class="stat-number">%d</span>
                <div class="stat-label">Categories</div>
            </div>
            <div class="stat-card">
                <span class="stat-number">%d</span>
                <div class="stat-label">Avg. Hot Score</div>
            </div>
            <div class="stat-card">
                <span class="stat-number">%.1f</span>
                <div class="stat-label">Hours Ago</div>
            </div>
        </section>

        <main>`, len(nr.Articles), len(categoryStats), avgScore, hoursAgo)

	// Group articles by category
	categorizedArticles := make(map[string][]NewsArticle)
	for _, article := range nr.Articles {
		categorizedArticles[article.Category] = append(categorizedArticles[article.Category], article)
	}

	// Category information
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
		htmlContent += fmt.Sprintf(`
            <section class="category-section" id="category-%s">
                <div class="category-header category-%s" onclick="toggleCategory('%s')">
                    <div class="category-title">
                        <span class="category-icon">%s</span>
                        <span>%s</span>
                        <div class="category-count">%d articles</div>
                        <span class="category-toggle">▼</span>
                    </div>
                </div>
                <div class="articles-grid">`, category, category, category, info.icon, info.name, len(articles))

		// Add articles for this category
		for _, article := range articles {
			timeAgo := formatTimeAgo(article.PublishedAt)

			// Generate keyword tags
			keywordTags := ""
			for _, keyword := range article.Keywords {
				if len(strings.TrimSpace(keyword)) > 0 {
					keywordTags += fmt.Sprintf(`<span class="keyword-tag">%s</span>`, html.EscapeString(keyword))
				}
			}

			htmlContent += fmt.Sprintf(`
                    <article class="article-card article-%s">
                        <h2 class="article-title">
                            <a href="%s" target="_blank">%s</a>
                        </h2>
                        <div class="article-meta">
                            <div>
                                <span class="article-source">%s</span>
                                <span class="article-time"> • %s</span>
                            </div>
                            <div class="hot-score">🔥 %d</div>
                        </div>
                        <p class="article-description">
                            %s
                        </p>
                        <div class="keywords">
                            %s
                        </div>
                    </article>`,
				category,
				html.EscapeString(article.URL),
				html.EscapeString(article.Title),
				html.EscapeString(article.Source),
				timeAgo,
				article.Score,
				html.EscapeString(article.Description),
				keywordTags)
		}

		htmlContent += `
                </div>
            </section>`
	}

	// Add footer and JavaScript
	htmlContent += `
        </main>

        <footer class="footer">
            <div class="footer-logo">Advanced Tech News Collector</div>
            <div class="footer-time">Report generated at ` + time.Now().Format("15:04:05 MST") + `</div>
			<div class="footer-copyright">© 2025 Thunder (thd3r)</div>
        </footer>
    </div>

    <script>
        // Category accordion functionality
        function toggleCategory(categoryId) {
            const categorySection = document.getElementById('category-' + categoryId);
            categorySection.classList.toggle('collapsed');
        }

        // Collapse all categories initially (optional)
        function initCategories() {
            // Uncomment the lines below if you want all categories collapsed by default
            const categories = document.querySelectorAll('.category-section');
            categories.forEach(category => category.classList.add('collapsed'));
        }

        // Theme toggle functionality
        function toggleTheme() {
            const body = document.body;
            const themeIcon = document.querySelector('.theme-icon');
            const themeText = document.querySelector('.theme-text');
            
            if (body.getAttribute('data-theme') === 'light') {
                body.removeAttribute('data-theme');
                themeIcon.textContent = '🌙';
                themeText.textContent = 'Dark';
                localStorage.setItem('theme', 'dark');
            } else {
                body.setAttribute('data-theme', 'light');
                themeIcon.textContent = '☀️';
                themeText.textContent = 'Light';
                localStorage.setItem('theme', 'light');
            }
        }

        // Initialize theme on page load
        function initTheme() {
            const savedTheme = localStorage.getItem('theme');
            const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
            const body = document.body;
            const themeIcon = document.querySelector('.theme-icon');
            const themeText = document.querySelector('.theme-text');
            
            if (savedTheme === 'light' || (!savedTheme && !prefersDark)) {
                body.setAttribute('data-theme', 'light');
                themeIcon.textContent = '☀️';
                themeText.textContent = 'Light';
            } else {
                body.removeAttribute('data-theme');
                themeIcon.textContent = '🌙';
                themeText.textContent = 'Dark';
            }
        }

        // Listen for system theme changes
        window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
            if (!localStorage.getItem('theme')) {
                initTheme();
            }
        });

        // Initialize everything when page loads
        document.addEventListener('DOMContentLoaded', () => {
            initTheme();
            initCategories();
        });

        // Keyboard shortcuts
        document.addEventListener('keydown', (e) => {
            // Theme toggle: Ctrl/Cmd + Shift + T
            if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key === 'T') {
                e.preventDefault();
                toggleTheme();
            }
            
            // Collapse all categories: Ctrl/Cmd + Shift + C
            if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key === 'C') {
                e.preventDefault();
                const categories = document.querySelectorAll('.category-section');
                categories.forEach(category => category.classList.add('collapsed'));
            }
            
            // Expand all categories: Ctrl/Cmd + Shift + E
            if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key === 'E') {
                e.preventDefault();
                const categories = document.querySelectorAll('.category-section');
                categories.forEach(category => category.classList.remove('collapsed'));
            }
        });
    </script>
</body>
</html>`

	return htmlContent
}

// formatTimeAgo formats time duration to human readable string
func formatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes <= 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
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

	report.WriteString("*Report generated by Advanced Tech News Collector By Thunder (thd3r)*\n")
	return report.String()
}

func (nr *NewsReporter) GenerateTwitterTxtReport(posts []map[string]interface{}) string {
	var report strings.Builder

	report.WriteString("=================================================================\n")
	report.WriteString("            🐦 TWITTER POSTS - READY TO USE\n")
	report.WriteString("=================================================================\n")
	report.WriteString(fmt.Sprintf("Generated: %s\n", time.Now().Format("Monday, January 2, 2006 at 15:04:05")))
	report.WriteString(fmt.Sprintf("Total Posts: %d\n", len(posts)))
	report.WriteString("=================================================================\n\n")

	for i, post := range posts {
		report.WriteString(fmt.Sprintf("POST #%d\n", i+1))
		report.WriteString(strings.Repeat("-", 50) + "\n")
		report.WriteString(fmt.Sprintf("Category: %s\n", post["category"]))
		report.WriteString(fmt.Sprintf("Score: %v\n", post["score"]))
		report.WriteString(fmt.Sprintf("Source: %s\n", post["source"]))
		report.WriteString(strings.Repeat("-", 50) + "\n")
		report.WriteString("TWEET CONTENT:\n")
		report.WriteString(fmt.Sprintf("%s\n", post["content"]))
		report.WriteString(strings.Repeat("-", 50) + "\n")
		report.WriteString(fmt.Sprintf("Character Count: %d/280\n", len(post["content"].(string))))
		report.WriteString(strings.Repeat("=", 50) + "\n\n")
	}

	report.WriteString("=================================================================\n")
	report.WriteString("                    POSTING INSTRUCTIONS\n")
	report.WriteString("=================================================================\n")
	report.WriteString("1. Copy each tweet content above\n")
	report.WriteString("2. Paste directly to Twitter/X\n")
	report.WriteString("3. Schedule or post immediately\n")
	report.WriteString("4. Monitor engagement and respond to comments\n")
	report.WriteString("=================================================================\n")
	report.WriteString("Report generated by Advanced Tech News Collector\n")

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

	fmt.Println("🚀 Starting Advanced Digital News Collection...")
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
		fmt.Printf("🐦 Twitter JSON report saved: %s\n", twitterFile)
	}

	twitterTxtFile := fmt.Sprintf("twitter_posts_ready_%s.txt", timestamp)
	twitterTxtContent := reporter.GenerateTwitterTxtReport(twitterPosts)
	if err := os.WriteFile(twitterTxtFile, []byte(twitterTxtContent), 0644); err != nil {
		log.Printf("Error saving Twitter TXT report: %v", err)
	} else {
		fmt.Printf("🐦 Twitter TXT report saved: %s\n", twitterTxtFile)
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
