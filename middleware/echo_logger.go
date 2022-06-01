package middleware

import (
	"thebeast/utils"

	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/labstack/echo"
)

type EchoLogFields map[string]string

type (
	// LoggerConfig defines the config for Logger middleware.
	EchoLoggerConfig struct {
		// Log format which can be constructed using the following tags:
		//
		// - time_rfc3339
		// - id (Request ID - Not implemented)
		// - remote_ip
		// - uri
		// - host
		// - method
		// - path
		// - referer
		// - user_agent
		// - status
		// - latency (In microseconds)
		// - latency_human (Human readable)
		// - bytes_in (Bytes received)
		// - bytes_out (Bytes sent)
		// - header:<name>
		// - query:<name>
		// - form:<name>
		//
		// Example "${remote_ip} ${status}"
		//
		// Optional. Default value DefaultLoggerConfig.Format.
		Fields EchoLogFields `json:"fields"`

		// Output is a writer where logs are written.
		// Optional. Default value os.Stdout.
		Output *log.Entry
	}
)

var (
	// DefaultLoggerConfig is the default Logger middleware config.
	DefaultEchoLoggerConfig = EchoLoggerConfig{
		Fields: EchoLogFields{
			"time":          "time_rfc3339",
			"remote_ip":     "remote_ip",
			"host":          "host",
			"method":        "method",
			"uri":           "uri",
			"status":        "status",
			"latency":       "latency",
			"latency_human": "latency_human",
			"bytes_in":      "bytes_in",
			"bytes_out":     "bytes_out",
		},
		Output: utils.Logger.WithField("kind", "response_log"),
	}
)

// Logger returns a middleware that logs HTTP requests.
func EchoLogger() echo.MiddlewareFunc {
	return EchoLoggerWithConfig(DefaultEchoLoggerConfig)
}

func EchoLoggerWithConfig(config EchoLoggerConfig) echo.MiddlewareFunc {
	// Defaults
	if len(config.Fields) == 0 {
		config.Fields = DefaultEchoLoggerConfig.Fields
	}
	if config.Output == nil {
		config.Output = DefaultEchoLoggerConfig.Output
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			req := c.Request()
			res := c.Response()
			start := time.Now()
			if err = next(c); err != nil {
				c.Error(err)
			}
			stop := time.Now()

			var toLog = make(map[string]interface{}, len(config.Fields)+3)

			for k, v := range config.Fields {
				switch v {
				case "time_rfc3339":
					toLog[k] = time.Now().Format(time.RFC3339)
				case "remote_ip":
					ra := c.RealIP()
					toLog[k] = ra
				case "host":
					toLog[k] = req.Host
				case "uri":
					toLog[k] = req.RequestURI
				case "method":
					toLog[k] = req.Method
				case "path":
					p := req.URL.Path
					if p == "" {
						p = "/"
					}
					toLog[k] = p
				case "referer":
					toLog[k] = req.Referer()
				case "user_agent":
					toLog[k] = req.UserAgent()
				case "status":
					toLog[k] = res.Status
				case "durations":
					l := float64(stop.Sub(start).Nanoseconds()) / 1000000.0
					parsed := strconv.FormatFloat(l, 'f', 4, 64)
					toLog[k] = parsed
				case "duration":
					l := float64(stop.Sub(start).Nanoseconds()) / 1000000.0
					parsed := strconv.FormatFloat(l, 'f', 4, 64)
					f, _ := strconv.ParseFloat(parsed, 64)
					toLog[k] = f
				case "latency":
					l := stop.Sub(start).Nanoseconds() / 1000
					toLog[k] = strconv.FormatInt(l, 10)
				case "latency_human":
					toLog[k] = stop.Sub(start).String()
				case "bytes_in":
					b := req.Header.Get(echo.HeaderContentLength)
					if b == "" {
						b = "0"
					}
					toLog[k] = b
				case "bytes_out":
					toLog[k] = strconv.FormatInt(res.Size, 10)
				case "json":
					params := c.Get("BodyParams")
					if params != nil {
						// TODO: filter out base64 params
						b, _ := json.Marshal(params)
						rb64, _ := regexp.Compile("data:[^;]+;base64,[^\"]+")
						filtered := rb64.ReplaceAllString(string(b), "[FILTERED]")
						//rbpass, _ := regexp.Compile(`(?i)"(.*password.*)":"((\\"|[^"])*)"`)
						rbpass, _ := regexp.Compile(`(?i)"(((\\"|[^"])*)password((\\"|[^"])*))":"((\\"|[^"])*)"`)
						filtered = rbpass.ReplaceAllString(filtered, "\"$1\":\"[FILTERED]\"")
						//return w.Write([]byte(fmt.Sprintf("%+v", c.Get("Params"))))
						toLog[k] = filtered
					}
				case "params":
					allParams := c.QueryParams()
					formParams, err := c.FormParams()
					if err == nil {
						for k, v := range formParams {
							allParams[k] = v
						}
					}
					if len(allParams) > 0 {
						b, err := json.Marshal(allParams)
						if err == nil {
							toLog[k] = string(b)
							break
						}
					}
				case "view":
					viewTime := c.Get("ViewTime")
					if viewTime != nil {
						viewTimeS := viewTime.(string)
						if viewTimeS != "" {
							f, _ := strconv.ParseFloat(viewTimeS, 64)
							toLog[k] = f
						}
					}
				case "plugin_times":
					pluginTime := c.Get("PluginTime")
					if pluginTime != nil {
						parsed := strconv.FormatFloat(pluginTime.(float64), 'f', 4, 64)
						toLog[k] = parsed
					}
				case "plugin_time":
					pluginTime := c.Get("PluginTime")
					if pluginTime != nil {
						parsed := strconv.FormatFloat(pluginTime.(float64), 'f', 4, 64)
						f, _ := strconv.ParseFloat(parsed, 64)
						toLog[k] = f
					}
				default:
					switch {
					case strings.HasPrefix(v, "header:"):
						header := c.Request().Header.Get(v[7:])
						if len(header) > 0 {
							toLog[k] = header
						}
					case strings.HasPrefix(v, "context:"):
						cont := c.Get(v[8:])
						if cont != nil {
							contS := cont.(string)
							toLog[k] = contS
						}

					case strings.HasPrefix(v, "query:"):
						toLog[k] = c.QueryParam(v[6:])
					case strings.HasPrefix(v, "form:"):
						toLog[k] = c.FormValue(v[5:])
					}
				}
			}
			toWrite := config.Output.WithFields(toLog)
			switch {
			case res.Status >= 500:
				toWrite.Error()
			case res.Status >= 400:
				toWrite.Warn()
			default:
				toWrite.Info()
			}
			return
		}
	}
}
