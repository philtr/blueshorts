package config

import "github.com/BurntSushi/toml"

// IMAPConfig is the section of the TOML file holding IMAP creds
// (kept here so the whole config parses in one shot).
type IMAPConfig struct {
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	Username string `toml:"username"`
	Password string `toml:"password"`
}

type Config struct {
	Server struct {
		Port   int    `toml:"port"`
		APIKey string `toml:"api_key"`
	} `toml:"server"`

	IMAP  IMAPConfig        `toml:"imap"`
	Feeds map[string]string `toml:"feeds"`
}

func Load(path string) (*Config, error) {
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
