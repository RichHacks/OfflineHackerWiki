package config

import (
	"flag"
	"path/filepath"
)

type Config struct {
	Port                int
	HackTricksPath      string
	RevShellsPath       string
	XSSCheatsheetPath   string
	CertipyPath         string
	PayloadsPath        string
	CyberChefPath       string
}

func Parse(contentBase string) *Config {
	cfg := &Config{}

	flag.IntVar(&cfg.Port, "port", 8888, "Port to listen on")
	flag.StringVar(&cfg.HackTricksPath, "hacktricks", joinBase(contentBase, "hacktricks"), "Path to HackTricks repo")
	flag.StringVar(&cfg.RevShellsPath, "revshells", joinBase(contentBase, "reverse-shell-generator"), "Path to reverse-shell-generator repo")
	flag.StringVar(&cfg.XSSCheatsheetPath, "xss", joinBase(contentBase, "xss-cheatsheet"), "Path to xss-cheatsheet repo")
	flag.StringVar(&cfg.CertipyPath, "certipy", joinBase(contentBase, "certipy"), "Path to Certipy wiki repo")
	flag.StringVar(&cfg.PayloadsPath, "payloads", joinBase(contentBase, "payloadsallthethings"), "Path to PayloadsAllTheThings repo")
	flag.StringVar(&cfg.CyberChefPath, "cyberchef", joinBase(contentBase, "cyberchef"), "Path to CyberChef static build (gh-pages)")

	flag.Parse()
	return cfg
}

func joinBase(base, name string) string {
	if base == "" {
		return ""
	}
	return filepath.Join(base, name)
}
