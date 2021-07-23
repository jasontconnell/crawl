package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/jasontconnell/crawl/conf"
	"github.com/jasontconnell/crawl/data"
	"github.com/jasontconnell/crawl/process"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	cfgfile := flag.String("c", "", "config file")
	timeout := flag.Int("timeout", 15, "timeout")
	retryLimit := flag.Int("retryLimit", 3, "retryLimit")
	flag.Parse()

	startTime := time.Now()
	baseUrl := ""
	if len(os.Args) == 2 {
		baseUrl = os.Args[1]
		if strings.Index(baseUrl, "http") == -1 {
			baseUrl = "https://" + baseUrl
		}
	} else if len(os.Args) == 1 {
		*cfgfile = "entrypoints.json"
	}

	cfg := conf.LoadConfig(*cfgfile, baseUrl)
	if cfg.Root == "" {
		log.Fatal("need a url")
	}

	hdr := make(data.Headers)
	for k, v := range cfg.Headers {
		hdr[k] = v
	}

	site, err := data.NewSite(cfg.Root, cfg.VirtualPaths, hdr, cfg.UrlsFile, cfg.ErrorsFile, *timeout, *retryLimit)
	if err != nil {
		log.Fatal("error initializing site", err)
	}
	defer site.CleanUp()

	process.Start(site)

	fmt.Println("\n\ndone. time", time.Since(startTime))
}
