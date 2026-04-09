package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DiscordToken      string
	GeminiAPIKey      string
	GeminiModel       string
	SilenceThreshold  time.Duration
	InterveneCooldown time.Duration
	CacheSize         int
	RoomBaseURL       string
	TargetTextChannel string
}

func Load() Config {
	// Best-effort local env loading; CI/production can still rely on real env vars.
	_ = godotenv.Load()

	c := Config{
		DiscordToken:      os.Getenv("DISCORD_BOT_TOKEN"),
		GeminiAPIKey:      os.Getenv("GEMINI_API_KEY"),
		GeminiModel:       getEnv("GEMINI_MODEL", "gemini-2.5-flash"),
		SilenceThreshold:  secondsToDuration(getEnv("SILENCE_SECONDS", "7"), 7),
		InterveneCooldown: secondsToDuration(getEnv("INTERVENE_COOLDOWN_SECONDS", "45"), 45),
		CacheSize:         getEnvInt("CACHE_SIZE", 64),
		RoomBaseURL:       getEnv("ROOM_BASE_URL", "http://localhost:3000/room"),
		TargetTextChannel: os.Getenv("TARGET_TEXT_CHANNEL"),
	}

	if c.DiscordToken == "" {
		log.Fatal("DISCORD_BOT_TOKEN is required")
	}

	return c
}

func getEnv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func secondsToDuration(raw string, fallback int) time.Duration {
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		n = fallback
	}
	return time.Duration(n) * time.Second
}
