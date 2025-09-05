package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ObiAU/hfnewsaggregator/internal/models"
)

type NewsAPIClient struct {
	apiKey string
	client *http.Client
}

type NewsAPIResponse struct {
	Status       string `json:"status"`
	TotalResults int    `json:"totalResults"`
	Articles     []struct {
		Source struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"source"`
		Author      string    `json:"author"`
		Title       string    `json:"title"`
		Description string    `json:"description"`
		URL         string    `json:"url"`
		URLToImage  string    `json:"urlToImage"`
		PublishedAt time.Time `json:"publishedAt"`
		Content     string    `json:"content"`
	} `json:"articles"`
}

func NewNewsAPIClient(apiKey string) *NewsAPIClient {
	return &NewsAPIClient{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *NewsAPIClient) FetchArticles(ctx context.Context, limit int) ([]models.Article, error) {
	url := fmt.Sprintf("https://newsapi.org/v2/everything?apiKey=%s&pageSize=%d&sortBy=publishedAt", c.apiKey, limit)

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
		return nil, fmt.Errorf("newsapi returned status %d", resp.StatusCode)
	}

	var apiResp NewsAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	if apiResp.Status != "ok" {
		return nil, fmt.Errorf("newsapi error: %s", apiResp.Status)
	}

	articles := make([]models.Article, 0, len(apiResp.Articles))
	for _, apiArticle := range apiResp.Articles {
		content := apiArticle.Description
		if content == "" {
			content = apiArticle.Content
		}

		article := models.Article{
			ID:          fmt.Sprintf("newsapi_%s", apiArticle.URL),
			Title:       apiArticle.Title,
			Content:     content,
			URL:         apiArticle.URL,
			Source:      "newsapi",
			PublishedAt: apiArticle.PublishedAt,
			Hash:        generateHash(apiArticle.Title + content),
			Metadata: map[string]string{
				"author":      apiArticle.Author,
				"source_name": apiArticle.Source.Name,
				"source_id":   apiArticle.Source.ID,
				"image_url":   apiArticle.URLToImage,
			},
		}
		articles = append(articles, article)
	}

	return articles, nil
}

func (c *NewsAPIClient) GetName() string {
	return "newsapi"
}
