package telegram

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/ObiAU/hfnewsaggregator/internal/models"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api        *tgbotapi.BotAPI
	webhookURL string
	userAlerts map[int64]*models.UserAlert
	mu         sync.RWMutex
}

func NewBot(token, webhookURL string) *Bot {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatalf("Failed to create telegram bot: %v", err)
	}

	return &Bot{
		api:        bot,
		webhookURL: webhookURL,
		userAlerts: make(map[int64]*models.UserAlert),
	}
}

func (b *Bot) Start(ctx context.Context) error {
	webhook, err := tgbotapi.NewWebhook(b.webhookURL)
	if err != nil {
		return err
	}

	_, err = b.api.Request(webhook)
	if err != nil {
		return err
	}

	info, err := b.api.GetWebhookInfo()
	if err != nil {
		return err
	}

	if info.LastErrorDate != 0 {
		log.Printf("Telegram webhook last error: %s", info.LastErrorMessage)
	}

	updates := b.api.ListenForWebhook("/webhook")

	go func() {
		for update := range updates {
			b.handleUpdate(ctx, update)
		}
	}()

	return nil
}

func (b *Bot) handleUpdate(ctx context.Context, update tgbotapi.Update) {
	if update.Message == nil {
		return
	}

	userID := update.Message.From.ID
	chatID := update.Message.Chat.ID
	text := update.Message.Text

	switch {
	case strings.HasPrefix(text, "/start"):
		b.handleStart(chatID)
	case strings.HasPrefix(text, "/alert"):
		b.handleAlertCommand(ctx, userID, chatID, text)
	case strings.HasPrefix(text, "/list"):
		b.handleListAlerts(chatID)
	case strings.HasPrefix(text, "/help"):
		b.handleHelp(chatID)
	default:
		b.handleUnknownCommand(chatID)
	}
}

func (b *Bot) handleStart(chatID int64) {
	msg := tgbotapi.NewMessage(chatID, `Welcome to HF News Aggregator! 📰

I'll send you personalized news alerts based on your preferences.

Commands:
/alert set category=politics keywords=bitcoin,crypto
/alert set category=technology tags=ai,blockchain
/list - View your current alerts
/help - Show this help message

Example alert formats:
• /alert set category=cryptocurrency
• /alert set keywords=bitcoin,ethereum,defi
• /alert set category=politics keywords=election,policy
• /alert set tags=ai,machine learning category=technology`)

	b.api.Send(msg)
}

func (b *Bot) handleAlertCommand(ctx context.Context, userID, chatID int64, text string) {
	parts := strings.Fields(text)
	if len(parts) < 3 {
		b.sendMessage(chatID, "Invalid alert format. Use: /alert set category=politics keywords=bitcoin,crypto")
		return
	}

	alert := &models.UserAlert{
		UserID:  userID,
		ChatID:  chatID,
		Enabled: true,
	}

	for _, part := range parts[2:] {
		if strings.Contains(part, "=") {
			kv := strings.SplitN(part, "=", 2)
			key, value := kv[0], kv[1]

			switch key {
			case "category":
				alert.Categories = append(alert.Categories, value)
			case "keywords":
				keywords := strings.Split(value, ",")
				alert.Keywords = append(alert.Keywords, keywords...)
			case "tags":
				tags := strings.Split(value, ",")
				alert.Tags = append(alert.Tags, tags...)
			}
		}
	}

	b.mu.Lock()
	b.userAlerts[userID] = alert
	b.mu.Unlock()

	response := fmt.Sprintf("Alert configured! 🎯\n\nCategories: %v\nKeywords: %v\nTags: %v",
		alert.Categories, alert.Keywords, alert.Tags)
	b.sendMessage(chatID, response)
}

func (b *Bot) handleListAlerts(chatID int64) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var userID int64
	for uid, alert := range b.userAlerts {
		if alert.ChatID == chatID {
			userID = uid
			break
		}
	}

	alert, exists := b.userAlerts[userID]
	if !exists {
		b.sendMessage(chatID, "No alerts configured. Use /alert set to create one.")
		return
	}

	response := fmt.Sprintf("Your current alerts: 📋\n\nCategories: %v\nKeywords: %v\nTags: %v\nStatus: %s",
		alert.Categories, alert.Keywords, alert.Tags,
		map[bool]string{true: "Enabled", false: "Disabled"}[alert.Enabled])
	b.sendMessage(chatID, response)
}

func (b *Bot) handleHelp(chatID int64) {
	helpText := `HF News Aggregator Help 📖

Commands:
/start - Welcome message and setup
/alert set [options] - Configure news alerts
/list - View your current alerts
/help - Show this help

Alert Configuration Options:
• category=politics - Filter by news category
• keywords=bitcoin,crypto - Filter by keywords
• tags=ai,blockchain - Filter by tags

Examples:
/alert set category=cryptocurrency
/alert set keywords=bitcoin,ethereum,defi
/alert set category=politics keywords=election,policy
/alert set tags=ai,machine learning category=technology

Categories: politics, technology, cryptocurrency, finance, sports, entertainment, health, science, world, business`

	b.sendMessage(chatID, helpText)
}

func (b *Bot) handleUnknownCommand(chatID int64) {
	b.sendMessage(chatID, "Unknown command. Use /help for available commands.")
}

func (b *Bot) SendAlert(ctx context.Context, article models.CategorizedArticle) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, alert := range b.userAlerts {
		if !alert.Enabled {
			continue
		}

		if b.matchesAlert(article, alert) {
			message := b.formatAlertMessage(article)
			b.sendMessage(alert.ChatID, message)
		}
	}
}

func (b *Bot) matchesAlert(article models.CategorizedArticle, alert *models.UserAlert) bool {
	for _, category := range alert.Categories {
		if strings.EqualFold(article.Category, category) {
			return true
		}
	}

	for _, keyword := range alert.Keywords {
		content := strings.ToLower(article.Title + " " + article.Content)
		if strings.Contains(content, strings.ToLower(keyword)) {
			return true
		}
	}

	for _, tag := range alert.Tags {
		for _, articleTag := range article.Tags {
			if strings.EqualFold(articleTag, tag) {
				return true
			}
		}
	}

	return false
}

func (b *Bot) formatAlertMessage(article models.CategorizedArticle) string {
	return fmt.Sprintf(`🚨 News Alert

📰 %s

📂 Category: %s
🏷️ Tags: %s
😊 Sentiment: %s
📊 Confidence: %.1f%%

📝 Summary: %s

🔗 Read more: %s

Source: %s`,
		article.Title,
		article.Category,
		strings.Join(article.Tags, ", "),
		article.Sentiment,
		article.Confidence*100,
		article.Summary,
		article.URL,
		article.Source)
}

func (b *Bot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.DisableWebPagePreview = true

	_, err := b.api.Send(msg)
	if err != nil {
		log.Printf("Failed to send telegram message: %v", err)
	}
}
