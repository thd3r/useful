package models

import "time"

// Config holds the application configuration
type Config struct {
	Sources        map[string]Source         `yaml:"sources"`
	Categories     map[string]CategoryFilter `yaml:"categories"`
	DetectorConfig DetectorConfig            `yaml:"detector"`
	SocialConfig   TwitterConfig             `yaml:"twitter"`
	ReporterConfig ReporterConfig            `yaml:"reporter"`
	MaxArticles    int                       `yaml:"max_articles"`
	RateLimit      int                       `yaml:"rate_limit"`
	Timeout        time.Duration             `yaml:"timeout"`
}

type DetectorConfig struct {
	ViralKeywords   []string           `yaml:"viral_keywords"`
	TrendingTopics  []string           `yaml:"trending_topics"`
	SourceWeights   map[string]float64 `yaml:"source_weights"`
	CategoryWeights map[string]float64 `yaml:"category_weights"`
	MinHotScore     int                `yaml:"min_hot_score"`
}

type TwitterConfig struct {
	Templates    map[string][]string `yaml:"templates"`
	MaxLength    int                 `yaml:"max_length"`
	HashtagLimit int                 `yaml:"hashtag_limit"`
}

type ReporterConfig struct {
	OutputDir    string   `yaml:"output_dir"`
	Formats      []string `yaml:"formats"`
	TemplatePath string   `yaml:"template_path"`
}
