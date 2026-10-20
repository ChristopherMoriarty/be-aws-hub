package config

import (
	"errors"
	"net"
	"net/url"
	"os"
	"time"
)

type Config struct {
	HTTPAddr    string
	DatabaseURL string
	LogLevel    string
}

func Load() (Config, error) {
	databaseURL, err := loadDatabaseURL()
	if err != nil {
		return Config{}, err
	}

	httpAddr := os.Getenv("HTTP_ADDR")
	if httpAddr == "" {
		httpAddr = ":8080"
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	return Config{
		HTTPAddr:    httpAddr,
		DatabaseURL: databaseURL,
		LogLevel:    logLevel,
	}, nil
}

const minJWTSecretLength = 16

type Auth struct {
	Secret     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

func LoadAuth() (Auth, error) {
	secret := os.Getenv("JWT_SECRET")
	if len(secret) < minJWTSecretLength {
		return Auth{}, errors.New("JWT_SECRET must be at least 16 characters")
	}

	accessTTL, err := durationEnv("JWT_ACCESS_TTL", 15*time.Minute)
	if err != nil {
		return Auth{}, err
	}
	refreshTTL, err := durationEnv("JWT_REFRESH_TTL", 7*24*time.Hour)
	if err != nil {
		return Auth{}, err
	}
	if refreshTTL <= accessTTL {
		return Auth{}, errors.New("JWT_REFRESH_TTL must be longer than JWT_ACCESS_TTL")
	}

	return Auth{
		Secret:     secret,
		AccessTTL:  accessTTL,
		RefreshTTL: refreshTTL,
	}, nil
}

func loadDatabaseURL() (string, error) {
	host := os.Getenv("DATABASE__HOST")
	if host == "" {
		if raw := os.Getenv("DATABASE_URL"); raw != "" {
			return raw, nil
		}
		return "", errors.New("DATABASE__HOST is not set")
	}

	password := os.Getenv("DATABASE__PASSWORD")
	if password == "" {
		return "", errors.New("DATABASE__PASSWORD is not set")
	}

	return postgresURL(
		host,
		getenv("DATABASE__PORT", "5432"),
		getenv("DATABASE__USER", "aws"),
		password,
		getenv("DATABASE__NAME", "aws_hub"),
	), nil
}

func postgresURL(host, port, user, password, name string) string {
	return (&url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(user, password),
		Host:     net.JoinHostPort(host, port),
		Path:     "/" + name,
		RawQuery: "sslmode=disable",
	}).String()
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func durationEnv(key string, fallback time.Duration) (time.Duration, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		return 0, errors.New(key + " must be a positive duration")
	}
	return value, nil
}
