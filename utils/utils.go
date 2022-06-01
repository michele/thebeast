package utils

import (
	"reflect"
	"strings"
	"time"
)

func CallFuncByName(any interface{}, name string, args ...interface{}) {
	inputs := make([]reflect.Value, len(args))
	for i, _ := range args {
		inputs[i] = reflect.ValueOf(args[i])
	}
	reflect.ValueOf(any).MethodByName(name).Call(inputs)
}

func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func CacheTimestamp(t time.Time) string {
	return RightPad(strings.Replace(t.UTC().Format("20060102150405.999999999"), ".", "", 1), "0", 23)
}

func RightPad(str string, with string, targetLen int) string {
	return str + strings.Repeat(with[0:1], targetLen-len(str))
}
