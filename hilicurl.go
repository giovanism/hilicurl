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
	records := make([]Record, 0, 10)
	for {
		select {
		case <-ctx.Done():
			fmt.Printf("--- GET %s statistics ---\n", url)
			printStatistics(records)
			return
		default:
			go func() {
				tCtx, cancel := context.WithTimeout(ctx, *timeout)
				defer cancel()
				res := request(tCtx, url)

				records = append(records, res)
			}()
			time.Sleep(*interval)
		}
	}
}

func request(ctx context.Context, url string) Record {
	var t3 time.Time
	rec := Record{}

	trace := &httptrace.ClientTrace{
		GotConn: func(_ httptrace.GotConnInfo) { t3 = time.Now() },
	}

	ctx = httptrace.WithClientTrace(ctx, trace)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	rec.Request = req

	res, err := http.DefaultClient.Do(req)
	rec.Response = res
	if err != nil {
		log.Printf("ERROR: %v", err)
		return rec
	}

	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Printf("ERROR: %v", err)
		return rec
	}

	t7 := time.Now()
	elapsed := t7.Sub(t3)

	log.Printf("%s: length=%d bytes time=%d ms\n", res.Status, len(bytes), elapsed.Milliseconds())

	rec.ElapsedTime = elapsed

	return rec
}

func printStatistics(records []Record) {
	nReq, nRes := len(records), 0

	for _, rec := range records {
		if rec.Response != nil {
			nRes++
		}
	}

	nTimeout := nReq - nRes
	timeoutRate := float64(nTimeout) / float64(nReq) * 100
	fmt.Printf("%d requests transmitted, %d responses received, %.2f%% timeout",
		nReq, nRes, timeoutRate)
}

type Record struct {
	Request     *http.Request
	Response    *http.Response
	ElapsedTime time.Duration
}
