package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

const defaultSleep = 2 * time.Second

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Error:", r)
			flag.Usage()
			os.Exit(1)
		}
	}()

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	setupCloseHandler(ctx, cancel)

	flag.Usage = func() {
		fmt.Printf("Usage: %s URL\n", os.Args[0])
		flag.PrintDefaults()
	}

	var help bool
	flag.BoolVar(&help, "help", false, "Print help")
	flag.BoolVar(&help, "h", false, "Shorthand for -help")
	flag.Parse()

	if help {
		flag.Usage()
		return
	}

	if flag.NArg() != 1 {
		log.Panic("url argument is required")
	}

	url := flag.Arg(0)
	log.Printf("GET %s\n", url)
	for {
		select {
		case <-ctx.Done():
			log.Println("Gracefully shutting down...")
			return
		default:
			request(url)
			time.Sleep(defaultSleep)
		}
	}
}

func setupCloseHandler(ctx context.Context, cancel func()) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	go func() {
		select {
		case <-c:
			log.Println("Ctrl+C pressed in Terminal")
			cancel()
		case <-ctx.Done():
			return
		}
	}()
}

func request(url string) {
	start := time.Now()
	res, err := http.Get(url)
	if err != nil {
		log.Printf("ERROR: %v", err)
	}
	elapsed := time.Since(start)

	log.Printf("%s: time=%d ms\n", res.Status, elapsed.Milliseconds())
}
