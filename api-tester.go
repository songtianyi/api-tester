package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/songtianyi/rrframework/logs"
	"github.com/songtianyi/rrframework/utils"
	"io/ioutil"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

var (
	_ = flag.String("uri", "", "uri to request, eg. http://10.9.101.194:8080/?xx=xx&xx=xx")
	_ = flag.String("method", "POST", "Http method")
	_ = flag.String("p", "", "Post file path")
	_ = flag.Int("c", 1, "Number of multiple requests to make at a time")
	_ = flag.Int("n", 1, "Number of requests to perform")
	_ = flag.String("T", "image/jpeg", "Content-Type")
	_ = flag.Bool("strict", false, "Strict mode, if it's true tester will exit when any request fail")
)

// stat vars
var (
	totalCost = uint64(0)
	maxCost   = uint64(0)
	minCost   = uint64(1<<32 - 1)
	success   = uint64(0)
	failed    = uint64(0)
)

func printStat() {
	// print stat
	tick := time.Tick(2 * time.Second)
	for {
		select {
		case <-tick:
			if success < 1 {
				break
			}
			logs.Info("QPS", success*1000000/totalCost, "TotalCost", totalCost, "AvgCost", totalCost/success, "MaxCost", maxCost, "MinCost", minCost,
				"Success", success, "Failed", failed)
		}
	}
}

func main() {

	// prepare
	c, _ := rrutils.FlagGetInt("c")
	n, _ := rrutils.FlagGetInt("n")

	var (
		sema   = make(chan struct{}, c)
		bucket = make(chan struct{}, n)
		pdata  []byte
		err    error
	)

	if !rrutils.FlagIsSet("uri") {
		rrutils.FlagHelp()
		return
	}

	if rrutils.FlagIsSet("p") {
		path, _ := rrutils.FlagGetString("p")
		pdata, err = ioutil.ReadFile(path)
		if err != nil {
			logs.Error(err)
			return
		}
	}

	med, _ := rrutils.FlagGetString("method")
	uri, _ := rrutils.FlagGetString("uri")
	cot, _ := rrutils.FlagGetString("T")
	strict, _ := rrutils.FlagGetBool("strict")

	// common http client
	client := &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives:   false,
			MaxIdleConnsPerHost: 40,
		},
	}

	// stat
	go printStat()

	// dispatch
	tick := time.Tick(2 * time.Second)
loop1:
	for {
		select {
		case bucket <- struct{}{}:
			// write ok, still has some requests to make

			// try write
			sema <- struct{}{}

			go func(sema chan struct{}) {
				// make request
				req, _ := http.NewRequest(med, uri, bytes.NewReader(pdata))
				req.Header.Add("Content-Type", cot)

				// do request
				start := time.Now().UnixNano()
				err := do(client, req, sema)
				end := time.Now().UnixNano()

				// stat
				if err != nil {
					atomic.AddUint64(&failed, 1)
					logs.Error(err)
					if strict {
						os.Exit(0)
					}
					return
				}
				atomic.AddUint64(&success, 1)
				cost := uint64((end - start) / 1000000)
				if cost < minCost {
					minCost = cost
				}
				if cost > maxCost {
					maxCost = cost
				}
				atomic.AddUint64(&totalCost, cost)
			}(sema)

		case <-tick:
			// bucket full, all requests dispatched
			if len(sema) < 1 {
				// all request returned
				logs.Info("Done")
				break loop1
			}
		}
	}
}

func do(client *http.Client, req *http.Request, sema chan struct{}) error {
	defer func() { <-sema }() // release
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("StatusCode %d, Body %s", resp.StatusCode, string(body))
	}
	return nil
}
