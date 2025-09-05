package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ObiAU/hfnewsaggregator/internal/models"
	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
)

type OpenAIClient struct {
	client *openai.Client
}

type CategorizationRequest struct {
	Articles []models.Article `json:"articles"`
}

type CategorizationResponse struct {
	Articles []CategorizedArticle `json:"articles"`
}

type CategorizedArticle struct {
	ID         string   `json:"id"`
	Category   string   `json:"category"`
	Tags       []string `json:"tags"`
	Sentiment  string   `json:"sentiment"`
	Summary    string   `json:"summary"`
	Confidence float64  `json:"confidence"`
}

func NewOpenAIClient(apiKey string) *OpenAIClient {
	client := openai.NewClient(option.WithAPIKey(apiKey))
	return &OpenAIClient{client: client}
}

func (c *OpenAIClient) CategorizeArticles(ctx context.Context, articles []models.Article) ([]models.CategorizedArticle, error) {
	if len(articles) == 0 {
		return nil, nil
	}

	prompt := c.buildCategorizationPrompt(articles)

	response, err := c.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: "gpt-4o-mini",
		Messages: []openai.ChatCompletionMessageParamUnion{
			{
				OfSystem: &openai.ChatCompletionSystemMessageParam{
					Content: "You are a news categorization expert. Analyze articles and provide structured categorization data.",
				},
			},
			{
				OfUser: &openai.ChatCompletionUserMessageParam{
					Content: openai.ChatCompletionUserMessageParamContentUnion{
						OfString: openai.String(prompt),
					},
				},
			},
		},
		Temperature: openai.Float64(0.1),
		MaxTokens:   openai.Int(4000),
	})

	if err != nil {
		return nil, fmt.Errorf("openai request failed: %w", err)
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no response from openai")
	}

	content := response.Choices[0].Message.Content
	var categorizationResp CategorizationResponse
	if err := json.Unmarshal([]byte(content), &categorizationResp); err != nil {
		return nil, fmt.Errorf("failed to parse openai response: %w", err)
	}

	categorized := make([]models.CategorizedArticle, 0, len(categorizationResp.Articles))
	for _, catArticle := range categorizationResp.Articles {
		categorized = append(categorized, models.CategorizedArticle{
			Article:     findArticleByID(articles, catArticle.ID),
			Confidence:  catArticle.Confidence,
			ProcessedAt: time.Now(),
		})
	}

	return categorized, nil
}

func (c *OpenAIClient) ValidateCategorization(ctx context.Context, article models.Article, category string) (bool, float64, error) {
	prompt := fmt.Sprintf(`
Article: %s
Content: %s
Assigned Category: %s

Does this article belong to the category "%s"? 
Respond with JSON: {"belongs": true/false, "confidence": 0.0-1.0, "reason": "brief explanation"}
`, article.Title, article.Content, category, category)

	response, err := c.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: "gpt-4o-mini",
		Messages: []openai.ChatCompletionMessageParamUnion{
			{
				OfUser: &openai.ChatCompletionUserMessageParam{
					Content: openai.ChatCompletionUserMessageParamContentUnion{
						OfString: openai.String(prompt),
					},
				},
			},
		},
		Temperature: openai.Float64(0.1),
		MaxTokens:   openai.Int(200),
	})

	if err != nil {
		return false, 0, err
	}

	if len(response.Choices) == 0 {
		return false, 0, fmt.Errorf("no response from openai")
	}

	content := response.Choices[0].Message.Content
	var validation struct {
		Belongs    bool    `json:"belongs"`
		Confidence float64 `json:"confidence"`
		Reason     string  `json:"reason"`
	}

	if err := json.Unmarshal([]byte(content), &validation); err != nil {
		return false, 0, err
	}

	return validation.Belongs, validation.Confidence, nil
}

func (c *OpenAIClient) buildCategorizationPrompt(articles []models.Article) string {
	var sb strings.Builder
	sb.WriteString("Categorize these news articles. For each article, provide:\n")
	sb.WriteString("- category: one of [politics, technology, cryptocurrency, finance, sports, entertainment, health, science, world, business]\n")
	sb.WriteString("- tags: relevant keywords (max 5)\n")
	sb.WriteString("- sentiment: positive, negative, or neutral\n")
	sb.WriteString("- summary: 1-2 sentence summary\n")
	sb.WriteString("- confidence: 0.0-1.0\n\n")
	sb.WriteString("Respond with JSON format:\n")
	sb.WriteString(`{"articles": [{"id": "article_id", "category": "category", "tags": ["tag1", "tag2"], "sentiment": "sentiment", "summary": "summary", "confidence": 0.95}]}`)
	sb.WriteString("\n\nArticles to categorize:\n\n")

	for i, article := range articles {
		sb.WriteString(fmt.Sprintf("Article %d:\n", i+1))
		sb.WriteString(fmt.Sprintf("ID: %s\n", article.ID))
		sb.WriteString(fmt.Sprintf("Title: %s\n", article.Title))
		sb.WriteString(fmt.Sprintf("Content: %s\n", article.Content))
		sb.WriteString(fmt.Sprintf("Source: %s\n", article.Source))
		sb.WriteString("\n")
	}

	return sb.String()
}

func findArticleByID(articles []models.Article, id string) models.Article {
	for _, article := range articles {
		if article.ID == id {
			return article
		}
	}
	return models.Article{}
}
