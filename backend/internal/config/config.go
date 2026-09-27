package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port             string
	Environment      string
	DatabaseURL      string
	JWTSecret        string
	WebOrigin        string
	StockfishPath    string
	AdminEmail       string
	AnalysisThreads  int
	AnalysisHashMB   int
	AnalysisWorkers  int
	QueueCapacity    int
	AnalysisMoveTime time.Duration
}

func Load() Config {
	loadDotEnv()
	return Config{
		Port:             env("API_PORT", "8080"),
		Environment:      env("APP_ENV", "development"),
		DatabaseURL:      env("DATABASE_URL", "file:royal_move.db?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"),
		JWTSecret:        env("JWT_SECRET", "development-secret-change-before-production"),
		WebOrigin:        env("WEB_ORIGIN", "http://localhost:3000"),
		StockfishPath:    env("STOCKFISH_PATH", defaultStockfishPath()),
		AdminEmail:       strings.ToLower(strings.TrimSpace(env("ADMIN_EMAIL", ""))),
		AnalysisThreads:  envInt("ANALYSIS_THREADS", 2),
		AnalysisHashMB:   envInt("ANALYSIS_HASH_MB", 256),
		AnalysisWorkers:  envInt("ANALYSIS_WORKERS", 1),
		QueueCapacity:    envInt("ANALYSIS_QUEUE_CAPACITY", 256),
		AnalysisMoveTime: time.Duration(envInt("ANALYSIS_MOVETIME_MS", 800)) * time.Millisecond,
	}
}

func loadDotEnv() {
	for _, path := range []string{".env", "../.env"} {
		file, err := os.Open(path)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			key, value, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			key = strings.TrimSpace(key)
			value = strings.Trim(strings.TrimSpace(value), "\"'")
			if key != "" && os.Getenv(key) == "" {
				_ = os.Setenv(key, value)
			}
		}
		_ = file.Close()
		return
	}
}

func (c Config) Validate() error {
	if c.Port == "" {
		return fmt.Errorf("API_PORT must not be empty")
	}
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL must not be empty")
	}
	if c.WebOrigin == "" {
		return fmt.Errorf("WEB_ORIGIN must not be empty")
	}
	if c.Environment == "production" && (len(c.JWTSecret) < 32 || strings.Contains(c.JWTSecret, "development-secret")) {
		return fmt.Errorf("JWT_SECRET must contain at least 32 secure characters in production")
	}
	if c.AnalysisThreads < 1 || c.AnalysisHashMB < 16 || c.AnalysisWorkers < 1 || c.QueueCapacity < 1 {
		return fmt.Errorf("analysis resource settings are invalid")
	}
	return nil
}

func defaultStockfishPath() string {
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "stockfish.exe")
		if _, err = os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return filepath.FromSlash("../tools/stockfish/stockfish/stockfish-windows-x86-64-universal.exe")
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value, err := strconv.Atoi(env(key, ""))
	if err != nil {
		return fallback
	}
	return value
}
