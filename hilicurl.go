package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptrace"
	"os"
	"os/signal"
	"time"
)

const (
	defaultInterval = 2 * time.Second
	defaultTimeout  = 60 * time.Second
)

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

	interval := flag.Duration("interval", defaultInterval, "Interval between each request")
	timeout := flag.Duration("timeout", defaultTimeout, "Request timeout")
	flag.Parse()

	if help {
		flag.Usage()
		return
	}

	if flag.NArg() != 1 {
		log.Panic("url argument is required")
	}

	url := flag.Arg(0)
	runRequests(ctx, url, interval, timeout)
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

func runRequests(ctx context.Context, url string, interval *time.Duration, timeout *time.Duration) {
	log.Printf("GET %s\n", url)
	responses := make([]*http.Response, 10)
	for {
		select {
		case <-ctx.Done():
			fmt.Printf("--- GET %s statistics ---\n", url)
			printStatistics(responses)
			return
		default:
			go func() {
				tCtx, cancel := context.WithTimeout(ctx, *timeout)
				defer cancel()
				res := request(tCtx, url)

				responses = append(responses, res)
			}()
			time.Sleep(*interval)
		}
	}
}

func request(ctx context.Context, url string) *http.Response {
	var t3 time.Time

	trace := &httptrace.ClientTrace{
		GotConn: func(_ httptrace.GotConnInfo) { t3 = time.Now() },
	}

	ctx = httptrace.WithClientTrace(ctx, trace)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("ERROR: %v", err)
		return nil
	}

	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Printf("ERROR: %v", err)
		return nil
	}

	t7 := time.Now()
	elapsed := t7.Sub(t3)

	log.Printf("%s: length=%d bytes time=%d ms\n", res.Status, len(bytes), elapsed.Milliseconds())
	return res
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
