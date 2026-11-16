package config

import (
	"fmt"
	"os"
)

type Config struct {
	ListenAddr        string
	DBPath            string
	Namespace         string
	CABundleNamespace string
	CABundleName      string
	CABundleKey       string
}

func FromEnv() (*Config, error) {
	cfg := &Config{
		ListenAddr:        envOr("PORTAL_LISTEN_ADDR", ":8080"),
		DBPath:            envOr("PORTAL_DB_PATH", "/data/portal.db"),
		Namespace:         envOr("PORTAL_NAMESPACE", "wildwest"),
		CABundleNamespace: os.Getenv("PORTAL_CABUNDLE_NAMESPACE"),
		CABundleName:      os.Getenv("PORTAL_CABUNDLE_NAME"),
		CABundleKey:       os.Getenv("PORTAL_CABUNDLE_KEY"),
	}
	if cfg.DBPath == "" {
		return nil, fmt.Errorf("PORTAL_DB_PATH is empty")
	}
	return cfg, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
