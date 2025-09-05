package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ObiAU/hfnewsaggregator/internal/models"
)

type CryptoPanicClient struct {
	apiKey string
	client *http.Client
}

type CryptoPanicResponse struct {
	Results []struct {
		ID        int    `json:"id"`
		Title     string `json:"title"`
		URL       string `json:"url"`
		Published string `json:"published_at"`
		Source    struct {
			Title string `json:"title"`
		} `json:"source"`
		Votes struct {
			Negative int `json:"negative"`
			Positive int `json:"positive"`
		} `json:"votes"`
	} `json:"results"`
}

func NewCryptoPanicClient(apiKey string) *CryptoPanicClient {
	return &CryptoPanicClient{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *CryptoPanicClient) FetchArticles(ctx context.Context, limit int) ([]models.Article, error) {
	url := fmt.Sprintf("https://cryptopanic.com/api/v1/posts/?auth_token=%s&public=true&page_size=%d", c.apiKey, limit)

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
		return nil, fmt.Errorf("cryptopanic returned status %d", resp.StatusCode)
	}

	var apiResp CryptoPanicResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	articles := make([]models.Article, 0, len(apiResp.Results))
	for _, result := range apiResp.Results {
		publishedAt, _ := time.Parse(time.RFC3339, result.Published)

		article := models.Article{
			ID:          fmt.Sprintf("cryptopanic_%d", result.ID),
			Title:       result.Title,
			Content:     result.Title,
			URL:         result.URL,
			Source:      "cryptopanic",
			PublishedAt: publishedAt,
			Hash:        generateHash(result.Title),
			Metadata: map[string]string{
				"source_name":    result.Source.Title,
				"votes_positive": fmt.Sprintf("%d", result.Votes.Positive),
				"votes_negative": fmt.Sprintf("%d", result.Votes.Negative),
			},
		}
		articles = append(articles, article)
	}

	return articles, nil
}

func (c *CryptoPanicClient) GetName() string {
	return "cryptopanic"
}
