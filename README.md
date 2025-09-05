# HF News Aggregator

Go news aggregator that continuously streams news from multiple news sources/APIs, categorizes & summarizes items with AI, and sends personalized alerts via Telegram based on user preferences.

## Features

- Multi-source news ingestion (NewsAPI, TreeNews, CryptoPanic, and more ... configurable)
- 24-hour cache with deduplication
- Telegram bot with configurable alerts
- Continuous streaming and processing
- Intel AI Agent for categorization and tagging
- Validator Agent for AI validation
- Memory cleanup (TTL based removal and eviction goroutines)

## Setup

```bash
go run main.go
```

## Config

```bash
OPENAI_API_KEY=sk-key-here
TELEGRAM_BOT_TOKEN=bot_token_here
TELEGRAM_WEBHOOK_URL=https://yourdomain.com/webhook
NEWS_API_KEY=newsapi_key
BATCH_SIZE=10
PROCESSING_INTERVAL=30s
CACHE_RETENTION=24h
SERVER_PORT=8080
```
**Note**: Keep batch size on the low end or run the risk of the model's context window not being able to handle as well. There are diminishing returns here.

**Note 2**: When hosting change `SERVER_PORT` appropriately. In a future update I will add in usage tracking and restrictions so you can increase AI inference interval as costs rack up.

## Usage

Configure alerts in Telegram:

```
/alert set category=cryptocurrency
/alert set keywords=bitcoin,ethereum
/alert set category=politics keywords=election,policy
```

Alert format:
```
🚨 News Alert
📰 Article Title
📂 Category: cryptocurrency
🏷️ Tags: bitcoin, defi
😊 Sentiment: positive
📊 Confidence: 95.0%
📝 Summary: Brief summary...
🔗 Read more: https://example.com
```
