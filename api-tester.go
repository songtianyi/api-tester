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
	if !rrutils.FlagIsSet("c") {
		logs.Error("c not set")
		return
	}

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

	method, _ := rrutils.FlagGetString("method")

	uri, _ := rrutils.FlagGetString("uri")
	ct, _ := rrutils.FlagGetString("T")

	// make request
	client := &http.Client{}

	// dispatch
	tick := time.Tick(2 * time.Second)
loop1:
	for {
		select {
		case bucket <- struct{}{}:
			// write ok, bucket not full
			// try write
			sema <- struct{}{}
			go func(sema chan struct{}) {
				defer func() { <-sema }() // release
				// do request
				req, _ := http.NewRequest(method, uri, bytes.NewReader(pdata))
				req.Header.Add("Content-Type", ct)
				// stat
				start := time.Now().UnixNano()
				err := do(client, req)
				end := time.Now().UnixNano()
				logs.Debug("cost", (end-start)/1000000)
				if err != nil {
					logs.Error(err)
				}
			}(sema)
			// bucket full, job done
		case <-tick:
			if len(sema) < 1 {
				// done
				logs.Info("Done")
				break loop1
			}
		}
	}
}

func do(client *http.Client, req *http.Request) error {
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
