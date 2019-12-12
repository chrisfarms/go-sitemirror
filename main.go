package main

import (
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/alphagov/spotlight-gel/cacher"
	"github.com/alphagov/spotlight-gel/engine"
)

func port() int64 {
	p := os.Getenv("PORT")
	if p == "0" {
		p = "8080"
	}
	n, _ := strconv.Atoi(p)
	return int64(n)
}

func main() {
	serverConfig, err := engine.ParseConfig(os.Args[0], os.Args[1:], os.Stderr)
	if err != nil {
		os.Exit(1)
	}
	serverConfig.Crawler.NoProxy = true
	serverConfig.Crawler.AutoDownloadDepth = 0
	serverConfig.AutoEnqueueInterval = time.Duration(0)
	serverConfig.Port = port()
	server := engine.FromConfig(cacher.NewFs(), serverConfig)

	downloaderConfig, err := engine.ParseConfig(os.Args[0], os.Args[1:], os.Stderr)
	if err != nil {
		os.Exit(1)
	}
	downloaderConfig.Crawler.NoProxy = false
	downloader := engine.FromConfig(cacher.NewFs(), downloaderConfig)
	serverConfig.Port = 7531

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	for sig := range c {
		switch sig {
		case os.Interrupt:
			go server.Stop()     // FiXME: this effectively skpis drain, done because it waits for enqueue to finish
			go downloader.Stop() // FIXME: above
			os.Exit(0)
		}
	}
}
