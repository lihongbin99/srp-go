//go:build !windows || idea
// +build !windows idea

package logger

import (
	"fmt"
)

type log struct {
	name string
}

func NewLog(name string) *log {
	return &log{
		name: name,
	}
}

func (l *log) Error(message ...interface{}) {
	fmt.Printf("\033[31mError %s -> %s: %v\033[0m\n", getTime(), l.name, message)
}

func (l *log) Info(message ...interface{}) {
	fmt.Printf("\033[32mInfo %s -> %s: %v\033[0m\n", getTime(), l.name, message)
}

func (l *log) Warn(message ...interface{}) {
	fmt.Printf("\033[33mWarn %s -> %s: %v\033[0m\n", getTime(), l.name, message)
}

func (l *log) Debug(message ...interface{}) {
	if Debug {
		fmt.Printf("\033[34mDebug %s -> %s: %v\033[0m\n", getTime(), l.name, message)
	}
}

func (l *log) Trace(message ...interface{}) {
	if Trace {
		fmt.Printf("\033[36mTrace %s -> %s: %v\033[0m\n", getTime(), l.name, message)
	}
}
