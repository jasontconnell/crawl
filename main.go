package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
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
	flag.Parse()

	startTime := time.Now()
	baseUrl := ""
	if len(os.Args) == 2 {
		baseUrl = os.Args[1]
	} else if len(os.Args) == 1 {
		*cfgfile = "entrypoints.json"
	}

	cfg := conf.LoadConfig(*cfgfile, baseUrl)
	if cfg.Root == "" {
		log.Fatal("need a url")
	}

	site := data.Site{}
	site.Root = cfg.Root
	site.VirtualPaths = cfg.VirtualPaths
	site.Headers = make(data.Headers)

	var err error
	site.ErrorFile, err = os.OpenFile(cfg.ErrorsFile, os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		log.Fatal("couldn't open errors file", cfg.ErrorsFile, err)
	}

	defer site.ErrorFile.Close()
	site.ErrorFile.WriteString("Url, Referrer, Code\n")

	site.UrlsFile, err = os.OpenFile(cfg.UrlsFile, os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		log.Fatal("couldn't open urls file", cfg.ErrorsFile, err)
	}

	defer site.UrlsFile.Close()
	site.UrlsFile.WriteString("Url, Referrer\n")
	site.UrlsFile.WriteString(site.Root + ",[Root]\n")

	for k, v := range cfg.Headers {
		site.Headers[k] = v
	}

	process.Start(site)

	fmt.Println("\n\ndone. time", time.Since(startTime))
}
