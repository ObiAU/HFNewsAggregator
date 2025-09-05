package cache

import (
	"sync"
	"time"

	"github.com/ObiAU/hfnewsaggregator/internal/models"
)

type Cache struct {
	mu            sync.RWMutex
	articles      map[string]models.Article
	processed     map[string]time.Time
	retention     time.Duration
	cleanupTicker *time.Ticker
	stopChan      chan struct{}
}

func New(retention time.Duration) *Cache {
	c := &Cache{
		articles:  make(map[string]models.Article),
		processed: make(map[string]time.Time),
		retention: retention,
		stopChan:  make(chan struct{}),
	}

	c.cleanupTicker = time.NewTicker(1 * time.Hour)
	go c.cleanup()

	return c
}

func (c *Cache) AddArticle(article models.Article) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.articles[article.Hash] = article
	c.processed[article.Hash] = time.Now()
}

func (c *Cache) HasArticle(hash string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, exists := c.articles[hash]
	return exists
}

func (c *Cache) GetArticle(hash string) (models.Article, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	article, exists := c.articles[hash]
	return article, exists
}

func (c *Cache) GetUnprocessedArticles() []models.Article {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var unprocessed []models.Article
	for hash, article := range c.articles {
		if _, processed := c.processed[hash]; !processed {
			unprocessed = append(unprocessed, article)
		}
	}

	return unprocessed
}

func (c *Cache) MarkProcessed(hash string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.processed[hash] = time.Now()
}

func (c *Cache) cleanup() {
	for {
		select {
		case <-c.cleanupTicker.C:
			c.performCleanup()
		case <-c.stopChan:
			return
		}
	}
}

func (c *Cache) performCleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	cutoff := time.Now().Add(-c.retention)

	for hash, processedTime := range c.processed {
		if processedTime.Before(cutoff) {
			delete(c.articles, hash)
			delete(c.processed, hash)
		}
	}
}

func (c *Cache) Close() {
	c.cleanupTicker.Stop()
	close(c.stopChan)
}

func (c *Cache) Stats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"total_articles": len(c.articles),
		"processed":      len(c.processed),
		"retention":      c.retention.String(),
	}
}
