// Copyright © 2017 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"math/rand"
	"thebeast/configuration"
	"thebeast/middleware"
	"thebeast/utils"

	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/labstack/echo"
	mdw "github.com/labstack/echo/middleware"
	requestid "github.com/michele/echo-requestid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var BatchSize int

var semLock = &sync.Mutex{}
var semaphores = map[string](chan int){}
var client = &http.Client{}

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: startServer,
}

func init() {
	RootCmd.AddCommand(startCmd)
	rand.Seed(time.Now().UnixNano())
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// startCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// startCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}

func startServer(cmd *cobra.Command, args []string) {
	r := echo.New()
	r.Use(mdw.Recover())
	// r.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
	//   Format: `{"response_log":{"request_id": "${header:X-Request-Id}", "time":"${time_rfc3339}",` +
	//     `"method":"${method}","uri":"${uri}","status":${status}, "duration":${duration},` +
	//     `"latency_human":"${latency_human}",` +
	//     `"params": ${params}, "json": ${json}, "account": ${header:X-Synapse-Data}}}` + "\n",
	// }))
	r.Use(middleware.EchoLoggerWithConfig(middleware.EchoLoggerConfig{
		Fields: map[string]string{
			"request_id":    "header:X-Request-Id",
			"method":        "method",
			"uri":           "uri",
			"status":        "status",
			"duration":      "duration",
			"latency_human": "latency_human",
		},
	}))
	r.Use(requestid.RequestIdMiddleware)

	r.POST("/", handler)

	r.Logger.Fatal(r.Start(":" + configuration.Config.GoPort))
}

func handler(c echo.Context) error {
	rid := c.Get("RequestId").(string)

	bodyBytes, _ := ioutil.ReadAll(c.Request().Body)

	resps, err := getResponses(bodyBytes, rid)

	if err != nil {
		utils.ErrorReqIdLog(rid, err)
		return c.NoContent(422)
	}

	c.Response().Header().Add("Content-Type", "application/json")
	json.NewEncoder(c.Response()).Encode(resps)
	//return c.JSON(200, resps)
	return nil
}

func getResponses(bodyBytes []byte, rid string) ([]Response, error) {
	var wg sync.WaitGroup
	var reqs []Request
	err := json.Unmarshal(bodyBytes, &reqs)

	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("Couldn't unmarshal body: %s", string(bodyBytes)))
		return []Response{}, err
	}

	totalReqs := len(reqs)
	doneReqs := 0
	var resp = make(chan Response, totalReqs)
	start := time.Now()
	utils.DebugReqIdLogf(rid, "Processing %d requests", totalReqs)
	utils.DebugReqIdLogf(rid, "Starting: %s", start)
	for {
		nextBatch := int(math.Min(float64(BatchSize), float64(totalReqs-doneReqs)))
		wg.Add(nextBatch)
		for i := doneReqs; i < doneReqs+nextBatch; i++ {
			utils.DebugReqIdLogf(rid, "Parsed Request: %+v", reqs[i])
			go makeCall(reqs[i], resp, &wg, rid)
		}
		wg.Wait()
		doneReqs = doneReqs + nextBatch
		if doneReqs >= totalReqs {
			stop := time.Now()
			utils.DebugReqIdLogf(rid, "All done: %s", stop)
			l := float64(stop.Sub(start).Nanoseconds()) / 1000000.0
			parsed := strconv.FormatFloat(l, 'f', 4, 64)
			utils.ReqIdLogf(rid, "Processed %d requests in %ss", totalReqs, parsed)
			break
		}
	}

	var resps []Response

	for i := 0; i < len(reqs); i++ {
		resps = append(resps, <-resp)
	}

	return resps, nil
}

func makeCall(req Request, ch chan Response, wg *sync.WaitGroup, rid string) {
	defer wg.Done()
	var theResponse Response

	theResponse.Id = req.Id

	url, err := url.ParseRequestURI(req.Uri)
	if err != nil {
		utils.ErrorReqIdLog(rid, err)
		ch <- Response{
			Id:     req.Id,
			Status: 422,
		}
		return
	}
	semLock.Lock()
	sem, ok := semaphores[url.Hostname()]
	if !ok {
		sem = make(chan int, configuration.Config.CallsPerHost)
		semaphores[url.Hostname()] = sem
	}
	semLock.Unlock()
	r, err := http.NewRequest(req.Method, fmt.Sprintf("%s://%s%s", url.Scheme, url.Host, url.Path), bytes.NewReader([]byte(req.Body)))
	if len(req.Headers) > 0 {
		for key, value := range req.Headers {
			r.Header[key] = []string{value}
		}
	}

	r.URL.RawQuery = url.RawQuery

	utils.DebugReqIdLogf(rid, "net/http Request: %+v", r)
	var resp *http.Response
	//fmt.Printf("Calling %s\n", req.Uri)
	err = retry(configuration.Config.MaxRetries, time.Duration(configuration.Config.RetryWait)*time.Second, func() error {
		var err error
		sem <- 1
		resp, err = client.Do(r)
		<-sem
		defer resp.Body.Close()
		if err != nil {
			utils.ErrorReqIdLog(rid, err)
			theResponse.Status = 502
			return err
		}

		theResponse.Status = resp.StatusCode
		theResponse.Headers = map[string]string{}

		for k, v := range resp.Header {
			theResponse.Headers[k] = v[0]
		}
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		theResponse.Body = string(bodyBytes)

		s := resp.StatusCode
		switch {
		case s >= 500:
			// Retry
			return fmt.Errorf("server error: %v", s)
		case s >= 400:
			// Don't retry, it was client's fault
			return stop{fmt.Errorf("client error: %v", s)}
		default:
			// Happy
			return nil
		}
	})

	if err != nil {
		utils.ErrorReqIdLog(rid, err)
	}

	ch <- theResponse

}

func retry(attempts int, sleep time.Duration, f func() error) error {
	if err := f(); err != nil {
		if s, ok := err.(stop); ok {
			// Return the original error for later checking
			return s.error
		}

		if attempts--; attempts > 0 {
			// Add some randomness to prevent creating a Thundering Herd
			jitter := time.Duration(rand.Int63n(int64(sleep)))
			sleep = sleep + jitter/2

			time.Sleep(sleep)
			return retry(attempts, 2*sleep, f)
		}
		return err
	}

	return nil
}

type stop struct {
	error
}

type Request struct {
	Id      string            `json:"id"`
	Uri     string            `json:"uri"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
	Method  string            `json:"method"`
}

type Response struct {
	Id      string            `json:"id"`
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}
