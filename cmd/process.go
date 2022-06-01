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
	"strconv"
	"thebeast/utils"
	"time"

	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// processCmd represents the process command
var processCmd = &cobra.Command{
	Use:   "process [JSON file]",
	Short: "Process requests from command line",
	Long:  `You can pass a JSON file with the requests to perform from the command line`,
	RunE:  processFile,
}

var rid string

func init() {
	RootCmd.AddCommand(processCmd)
	processCmd.PersistentFlags().StringVarP(&rid, "request-id", "r", "", "X-Request-Id for logging purposes")
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// processCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// processCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}

func processFile(cmd *cobra.Command, args []string) error {
	if rid == "" {
		rid = "INTERNAL"
	}
	var bodyBytes []byte
	var err error
	if len(args) < 1 {
		bodyBytes, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			utils.ErrorReqIdLog(rid, errors.New("Couldn't read from STDIN"))
			return errors.New("403")
		}
		if len(bodyBytes) == 0 {
			utils.ErrorReqIdLog(rid, errors.New("You need to pass the file with the JSON request to process"))
			return errors.New("401")
		}

	} else {
		bodyBytes, err = ioutil.ReadFile(args[0])
		if err != nil {
			utils.ErrorReqIdLog(rid, errors.New("Couldn't find file"))
			return errors.New("404")
		}
	}
	start := time.Now()
	resps, err := getResponses(bodyBytes, rid)
	stop := time.Now()

	utils.DebugReqIdLogf(rid, "All done: %s", stop)
	l := float64(stop.Sub(start).Nanoseconds()) / 1000000.0
	parsed := strconv.FormatFloat(l, 'f', 4, 64)
	f, _ := strconv.ParseFloat(parsed, 64)
	utils.Logger.WithField("kind", "beast_log").WithField("duration", f).WithField("request_id", rid).Info("Finished processig")

	if err != nil {
		err = errors.Wrap(err, "Couldn't process request")
		utils.ErrorReqIdLogf(rid, err, "Got error: %+v", err)
		return errors.New("422")
	}

	json.NewEncoder(os.Stdout).Encode(resps)

	//return c.JSON(200, resps)
	return nil
}
