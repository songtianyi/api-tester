package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/songtianyi/rrframework/logs"
	"github.com/songtianyi/rrframework/utils"
	"io/ioutil"
	"net/http"
	"time"
)

var (
	_ = flag.String("uri", "", "uri to request, eg. http://10.9.101.194:8080/?xx=xx&xx=xx")
	_ = flag.String("method", "POST", "Http method")
	_ = flag.String("p", "", "Post file path")
	_ = flag.Int("c", 1, "Number of multiple requests to make at a time")
	_ = flag.Int("n", 1, "Number of requests to perform")
	_ = flag.String("T", "image/jpeg", "Content-Type")
)

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

	// common http client
	client := &http.Client{}

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

				// stat and output
				logs.Debug("cost", (end-start)/1000000)
				if err != nil {
					logs.Error(err)
				}
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
	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()
		return fmt.Errorf("StatusCode %d, Body %s", resp.StatusCode, string(body))
	}
	return nil
}
