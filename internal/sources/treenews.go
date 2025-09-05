package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ObiAU/hfnewsaggregator/internal/models"
)

type TreeNewsClient struct {
	client *http.Client
}

type TreeNewsMessage struct {
	ID          string `json:"_id"`
	Title       string `json:"title"`
	Source      string `json:"source,omitempty"`
	URL         string `json:"url,omitempty"`
	RawTime     int64  `json:"time"`
	Suggestions []struct {
		Coin string `json:"coin"`
	} `json:"suggestions,omitempty"`
	Info map[string]any `json:"info,omitempty"`
}

func NewTreeNewsClient() *TreeNewsClient {
	return &TreeNewsClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *TreeNewsClient) FetchArticles(ctx context.Context, limit int) ([]models.Article, error) {
	url := "https://news.treeofalpha.com/api/news"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("treenews returned status %d", resp.StatusCode)
	}

	var messages []TreeNewsMessage
	if err := json.NewDecoder(resp.Body).Decode(&messages); err != nil {
		return nil, err
	}

	articles := make([]models.Article, 0, len(messages))
	for _, msg := range messages {
		if len(articles) >= limit {
			break
		}

		publishedAt := time.Unix(msg.RawTime/1000, (msg.RawTime%1000)*int64(time.Millisecond))

		coins := make([]string, len(msg.Suggestions))
		for i, s := range msg.Suggestions {
			coins[i] = s.Coin
		}

		metadata := map[string]string{
			"source": msg.Source,
		}
		if len(coins) > 0 {
			metadata["suggested_coins"] = fmt.Sprintf("%v", coins)
		}

		article := models.Article{
			ID:          msg.ID,
			Title:       msg.Title,
			Content:     msg.Title,
			URL:         msg.URL,
			Source:      "treenews",
			PublishedAt: publishedAt,
			Hash:        generateHash(msg.Title),
			Metadata:    metadata,
		}
		articles = append(articles, article)
	}

	return articles, nil
}

func (c *TreeNewsClient) GetName() string {
	return "treenews"
}
