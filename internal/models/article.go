package models

import (
	"context"
	"time"
)

type Article struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Content     string            `json:"content"`
	URL         string            `json:"url"`
	Source      string            `json:"source"`
	PublishedAt time.Time         `json:"published_at"`
	Hash        string            `json:"hash"`
	Category    string            `json:"category"`
	Tags        []string          `json:"tags"`
	Sentiment   string            `json:"sentiment"`
	Summary     string            `json:"summary"`
	Metadata    map[string]string `json:"metadata"`
}

type NewsSource interface {
	FetchArticles(ctx context.Context, limit int) ([]Article, error)
	GetName() string
}

type CategorizedArticle struct {
	Article
	Confidence  float64   `json:"confidence"`
	ProcessedAt time.Time `json:"processed_at"`
}

type UserAlert struct {
	UserID     int64    `json:"user_id"`
	ChatID     int64    `json:"chat_id"`
	Keywords   []string `json:"keywords"`
	Categories []string `json:"categories"`
	Tags       []string `json:"tags"`
	Enabled    bool     `json:"enabled"`
}
