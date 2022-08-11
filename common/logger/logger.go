package logger

import "time"

var (
	Debug = false
	Trace = false
)

func SetLogLevel(level string) {
	if level == "debug" {
		Debug = true
		Trace = false
	} else if level == "trace" {
		Debug = true
		Trace = true
	} else {
		Debug = false
		Trace = false
	}
}

func getTime() string {
	return time.Now().Format("2006:01:02 15:04:05")
}
