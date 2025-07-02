package models

import "time"

// NewsArticle represents a single news article
type NewsArticle struct {
	ID          string    `json:"id" yaml:"id"`
	Title       string    `json:"title" yaml:"title"`
	Description string    `json:"description" yaml:"description"`
	URL         string    `json:"url" yaml:"url"`
	Source      string    `json:"source" yaml:"source"`
	PublishedAt time.Time `json:"published_at" yaml:"published_at"`
	Category    string    `json:"category" yaml:"category"`
	Score       int       `json:"score" yaml:"score"`
	Keywords    []string  `json:"keywords" yaml:"keywords"`
}

// CategoryFilter defines advanced filtering rules for each category
type CategoryFilter struct {
	PrimaryKeywords   []string
	SecondaryKeywords []string
	ExcludeKeywords   []string
	MinScore          int
	WeightMultiplier  float64
}

// Source represents a news source configuration
type Source struct {
	Name        string            `yaml:"name"`
	Enabled     bool              `yaml:"enabled"`
	RateLimit   int               `yaml:"rate_limit"`
	Timeout     time.Duration     `yaml:"timeout"`
	Credentials map[string]string `yaml:"credentials"`
}
