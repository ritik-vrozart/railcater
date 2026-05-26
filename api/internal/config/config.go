package config

import (
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	DatabaseURL string
	TenantID    uuid.UUID
	CORSOrigins []string
	HTTPTimeout time.Duration
	JWTSecret   string
	JWTExpiry   time.Duration
	AgentNotifyURL    string
	AgentNotifySecret string
}

func Load() (Config, error) {
	_ = godotenv.Load()

	port := getenv("PORT", "")
	dbURL := getenv("DATABASE_URL", "")
	tenantID, err := uuid.Parse(getenv("DEFAULT_TENANT_ID", "00000000-0000-0000-0000-000000000001"))
	if err != nil {
		return Config{}, err
	}

	timeoutSec, _ := strconv.Atoi(getenv("HTTP_TIMEOUT_SEC", "30"))
	if timeoutSec <= 0 {
		timeoutSec = 30
	}

	cors := getenv("CORS_ORIGINS", "")
	jwtSecret := getenv("JWT_SECRET", "")
	jwtHours, _ := strconv.Atoi(getenv("JWT_EXPIRY_HOURS", "72"))
	if jwtHours <= 0 {
		jwtHours = 72
	}

	return Config{
		Port:              port,
		DatabaseURL:       dbURL,
		TenantID:          tenantID,
		CORSOrigins:       splitCSV(cors),
		HTTPTimeout:       time.Duration(timeoutSec) * time.Second,
		JWTSecret:         jwtSecret,
		JWTExpiry:         time.Duration(jwtHours) * time.Hour,
		AgentNotifyURL:    getenv("AGENT_NOTIFY_URL", ""),
		AgentNotifySecret: getenv("AGENT_NOTIFY_SECRET", ""),
	}, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func splitCSV(s string) []string {
	var out []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			part := trimSpace(s[start:i])
			if part != "" {
				out = append(out, part)
			}
			start = i + 1
		}
	}
	return out
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}
