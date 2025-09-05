package aggregator

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/ObiAU/hfnewsaggregator/internal/ai"
	"github.com/ObiAU/hfnewsaggregator/internal/cache"
	"github.com/ObiAU/hfnewsaggregator/internal/config"
	"github.com/ObiAU/hfnewsaggregator/internal/models"
	"github.com/ObiAU/hfnewsaggregator/internal/sources"
	"github.com/ObiAU/hfnewsaggregator/internal/telegram"
)

type Aggregator struct {
	config      *config.Config
	cache       *cache.Cache
	telegramBot *telegram.Bot
	aiClient    *ai.OpenAIClient
	sources     []models.NewsSource
	server      *http.Server
	mu          sync.RWMutex
	running     bool
	stopChan    chan struct{}
}

func New(cfg *config.Config, cacheLayer *cache.Cache, bot *telegram.Bot) *Aggregator {
	aiClient := ai.NewOpenAIClient(cfg.OpenAIAPIKey)

	newsSources := []models.NewsSource{
		sources.NewNewsAPIClient(cfg.NewsAPIKey),
		sources.NewTreeNewsClient(),
		sources.NewCryptoPanicClient(cfg.NewsAPIKey),
	}

	return &Aggregator{
		config:      cfg,
		cache:       cacheLayer,
		telegramBot: bot,
		aiClient:    aiClient,
		sources:     newsSources,
		stopChan:    make(chan struct{}),
	}
}

func (a *Aggregator) Run(ctx context.Context) error {
	a.mu.Lock()
	a.running = true
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.running = false
		a.mu.Unlock()
	}()

	if err := a.telegramBot.Start(ctx); err != nil {
		return fmt.Errorf("failed to start telegram bot: %w", err)
	}

	go a.startHTTPServer(ctx)
	go a.processNewsLoop(ctx)

	<-ctx.Done()
	return a.shutdown()
}

func (a *Aggregator) processNewsLoop(ctx context.Context) {
	ticker := time.NewTicker(a.config.ProcessingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := a.processNewsBatch(ctx); err != nil {
				log.Printf("Error processing news batch: %v", err)
			}
		}
	}
}

func (a *Aggregator) processNewsBatch(ctx context.Context) error {
	var allArticles []models.Article
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, source := range a.sources {
		wg.Add(1)
		go func(src models.NewsSource) {
			defer wg.Done()

			articles, err := src.FetchArticles(ctx, a.config.BatchSize)
			if err != nil {
				log.Printf("Error fetching from %s: %v", src.GetName(), err)
				return
			}

			mu.Lock()
			allArticles = append(allArticles, articles...)
			mu.Unlock()
		}(source)
	}

	wg.Wait()

	newArticles := a.filterNewArticles(allArticles)
	if len(newArticles) == 0 {
		return nil
	}

	log.Printf("Processing %d new articles", len(newArticles))

	categorized, err := a.aiClient.CategorizeArticles(ctx, newArticles)
	if err != nil {
		return fmt.Errorf("failed to categorize articles: %w", err)
	}

	for _, catArticle := range categorized {
		a.cache.AddArticle(catArticle.Article)
		a.cache.MarkProcessed(catArticle.Article.Hash)

		go a.telegramBot.SendAlert(ctx, catArticle)
	}

	return nil
}

func (a *Aggregator) filterNewArticles(articles []models.Article) []models.Article {
	var newArticles []models.Article

	for _, article := range articles {
		if !a.cache.HasArticle(article.Hash) {
			newArticles = append(newArticles, article)
		}
	}

	return newArticles
}

func (a *Aggregator) startHTTPServer(ctx context.Context) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", a.healthHandler)
	mux.HandleFunc("/stats", a.statsHandler)
	mux.HandleFunc("/webhook", a.telegramWebhookHandler)

	a.server = &http.Server{
		Addr:    ":" + a.config.ServerPort,
		Handler: mux,
	}

	go func() {
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()
}

func (a *Aggregator) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"healthy","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
}

func (a *Aggregator) statsHandler(w http.ResponseWriter, r *http.Request) {
	stats := a.cache.Stats()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"cache_stats":%+v,"running":%t}`, stats, a.isRunning())
}

func (a *Aggregator) telegramWebhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (a *Aggregator) isRunning() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.running
}

func (a *Aggregator) shutdown() error {
	log.Println("Shutting down aggregator...")

	if a.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := a.server.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown HTTP server: %w", err)
		}
	}

	close(a.stopChan)
	return nil
}
