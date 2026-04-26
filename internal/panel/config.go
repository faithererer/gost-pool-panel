package panel

import (
	"os"
	"path/filepath"
)

type Config struct {
	BaseURL       string
	Listen        string
	DataPath      string
	AdminUser     string
	AdminPassword string
	Secret        string
}

func LoadConfig() Config {
	port := getenv("PANEL_PORT", "3000")
	return Config{
		BaseURL:       getenv("PANEL_BASE_URL", "http://127.0.0.1:"+port),
		Listen:        getenv("PANEL_LISTEN", ":"+port),
		DataPath:      getenv("PANEL_DATA_PATH", filepath.Join("data", "state.json")),
		AdminUser:     getenv("PANEL_ADMIN_USER", "admin"),
		AdminPassword: getenv("PANEL_ADMIN_PASSWORD", "admin123"),
		Secret:        getenv("PANEL_SECRET", "change-me"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
