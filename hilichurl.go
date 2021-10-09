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
	runRequests(ctx, url)
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

func request(url string) *http.Response {
	start := time.Now()
	res, err := http.Get(url)
	if err != nil {
		log.Printf("ERROR: %v", err)
		return nil
	}
	elapsed := time.Since(start)

	log.Printf("%s: time=%d ms\n", res.Status, elapsed.Milliseconds())
	return res
}

func runRequests(ctx context.Context, url string) {
	log.Printf("GET %s\n", url)
	responses := make([]*http.Response, 10)
	for {
		select {
		case <-ctx.Done():
			fmt.Printf("--- GET %s statistics ---\n", url)
			printStatistics(responses)
			return
		default:
			res := request(url)
			responses = append(responses, res)
			time.Sleep(defaultSleep)
		}
	}
}

func printStatistics(responses []*http.Response) {
	nReq, nRes := len(responses), 0

	for _, res := range responses {
		if res != nil {
			nRes++
		}
	}

	nTout := nReq - nRes
	fmt.Printf("%d requests transmitted, %d responses received, %.2f%% timeout",
		   nReq, nRes, float64(nTout/nReq)*100)
}
