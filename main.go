package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/ObiAU/hfnewsaggregator/internal/aggregator"
	"github.com/ObiAU/hfnewsaggregator/internal/cache"
	"github.com/ObiAU/hfnewsaggregator/internal/config"
	"github.com/ObiAU/hfnewsaggregator/internal/telegram"
)

func main() {
	cfg := config.Load()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cacheLayer := cache.New(24 * time.Hour)
	defer cacheLayer.Close()

	telegramBot := telegram.NewBot(cfg.TelegramToken, cfg.TelegramWebhookURL)

	newsAggregator := aggregator.New(cfg, cacheLayer, telegramBot)

	log.Println("Starting HF News Aggregator...")
	newsAggregator.Run(ctx)
	log.Println("HF News Aggregator stopped gracefully")
}
