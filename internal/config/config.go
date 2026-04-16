package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	OutputDir string     `toml:"output_dir"`
	CSVPath   string     `toml:"csv_path"`
	Mail      MailConfig `toml:"mail"`
}

type MailConfig struct {
	SMTPHost string `toml:"smtp_host"`
	SMTPPort int    `toml:"smtp_port"`
	From     string `toml:"from"`
	Password string `toml:"password"`
	To       string `toml:"to"`
}

func Load() (*Config, error) {
	cfg := &Config{
		OutputDir: defaultOutputDir(),
		CSVPath:   defaultCSVPath(),
	}

	path := configPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := writeDefault(path, cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	}

	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, err
	}

	cfg.OutputDir = expandHome(cfg.OutputDir)
	cfg.CSVPath = expandHome(cfg.CSVPath)
	return cfg, nil
}

func configPath() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		dir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(dir, "mkepub", "config.toml")
}

func dataDir() string {
	dir := os.Getenv("XDG_DATA_HOME")
	if dir == "" {
		dir = filepath.Join(os.Getenv("HOME"), ".local", "share")
	}
	return filepath.Join(dir, "mkepub")
}

func defaultOutputDir() string {
	return filepath.Join(filepath.Dir(dataDir()), "epub")
}

func defaultCSVPath() string {
	return filepath.Join(dataDir(), "list_person_all_extended_utf8.csv")
}

func writeDefault(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	content := fmt.Sprintf(`# mkepub config

output_dir = %q
csv_path   = %q

[mail]
smtp_host = "smtp.gmail.com"
smtp_port = 587
from      = ""
password  = ""
to        = ""
`, cfg.OutputDir, cfg.CSVPath)
	return os.WriteFile(path, []byte(content), 0600)
}

func expandHome(path string) string {
	if len(path) >= 2 && path[:2] == "~/" {
		return filepath.Join(os.Getenv("HOME"), path[2:])
	}
	return path
}
