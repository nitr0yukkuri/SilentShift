package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"silentshift/internal/ai"
	"silentshift/internal/config"
	"silentshift/internal/discord"
	"silentshift/internal/logcache"
)

func main() {
	cfg := config.Load()

	cache := logcache.NewCache(cfg.CacheSize)
	gemini := ai.NewClient(cfg)
	bot := discord.NewBot(cfg, cache, gemini)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := bot.Start(ctx); err != nil {
		log.Fatalf("failed to start bot: %v", err)
	}

	<-ctx.Done()
	if err := bot.Close(); err != nil {
		log.Printf("close error: %v", err)
	}
}
