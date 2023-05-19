package conf

import (
	"github.com/jasontconnell/conf"
)

type Config struct {
	Root         string   `json:"root"`
	VirtualPaths []string `json:"virtualPaths"`
	ReplaceRoots []string `json:"replaceRoots"`
	ErrorsFile   string   `json:"errorsFile"`
	UrlsFile     string   `json:"urlsFile"`
	Headers      Headers  `json:"headers"`
	Sitemap      string   `json:"sitemap"`
}

type Headers map[string]string

func LoadConfig(filename, baseUrl string) Config {
	cfg := Config{}

	err := conf.LoadConfig(filename, &cfg)

	if err != nil {
		cfg.Root = baseUrl
		cfg.ErrorsFile = "errors.txt"
		cfg.UrlsFile = "urls.txt"
	}

	return cfg
}
