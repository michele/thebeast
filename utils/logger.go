package utils

import (
	c "thebeast/configuration"

	"fmt"

	log "github.com/Sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

var Logger *log.Entry

func init() {
	if c.Config.GoLogLevel != "debug" {
		log.SetLevel(log.InfoLevel)
	} else {
		log.SetLevel(log.DebugLevel)
	}

	if c.Config.GoEnv == "development" {
		log.SetFormatter(&prefixed.TextFormatter{})
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetFormatter(&CustomJSONFormatter{})
		//log.SetFormatter(&log.JSONFormatter{})
	}

	Logger = log.WithFields(log.Fields{
		"kind": "extra_log",
		"app":  c.Config.AppName,
	})
}

func InitLogger() *log.Entry {
	return Logger.WithField("kind", "init_log")
}
func contextLogger(rid string, err error) *log.Entry {
	tempLogger := Logger.WithFields(log.Fields{
		"request_id": rid,
	})
	if err != nil {
		tempLogger = tempLogger.WithError(err)
	}
	return tempLogger
}
func ReqIdLog(rid string, v ...interface{}) {
	contextLogger(rid, nil).Infoln(fmt.Sprint(v...))
}

func ReqIdLogf(rid string, format string, v ...interface{}) {
	contextLogger(rid, nil).Infoln(fmt.Sprintf(format, v...))
}

func DebugReqIdLog(rid string, v ...interface{}) {
	contextLogger(rid, nil).Debugln(fmt.Sprint(v...))
}

func DebugReqIdLogf(rid string, format string, v ...interface{}) {
	contextLogger(rid, nil).Debugln(fmt.Sprintf(format, v...))
}

func WarnReqIdLog(rid string, v ...interface{}) {
	contextLogger(rid, nil).Warnln(fmt.Sprint(v...))
}

func WarnReqIdLogf(rid string, format string, v ...interface{}) {
	contextLogger(rid, nil).Warnln(fmt.Sprintf(format, v...))
}

func ErrorReqIdLog(rid string, err error, v ...interface{}) {
	contextLogger(rid, err).Errorln(fmt.Sprint(v...))
}

func ErrorReqIdLogf(rid string, err error, format string, v ...interface{}) {
	contextLogger(rid, err).Errorln(fmt.Sprintf(format, v...))
}
