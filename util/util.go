package util

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/debug"
)

type logger struct {
	tag string
	log *log.Logger
}

func NewLogger(tag string) *logger {
	return &logger{tag: tag, log: log.New(os.Stdout, fmt.Sprintf("[%v] ", tag), log.LstdFlags)}
}
func (l *logger) SetTag(tag string) {
	l.tag = tag
}
func (l *logger) Log(msg interface{}) {
	l.log.Println("\u001B[32m" + fmt.Sprintf("%v", msg) + "\u001B[0m")
}
func (l *logger) Warn(msg interface{}) {
	l.log.Println("\u001B[33m" + fmt.Sprintf("%v", msg) + "\u001B[0m")
}
func (l *logger) Error(msg interface{}) {

	l.log.Println("\u001B[31m" + fmt.Sprintf("%v", msg) + "\u001B[0m")
}
func (l *logger) Panic(msg interface{}) {
	panic("\u001B[31m" + fmt.Sprintf("%v", msg) + "\u001B[0m")
}

var Log *logger = NewLogger("default")

func Recover() {
	if err := recover(); err != nil {
		_, file, line, ok := runtime.Caller(3)
		if ok {
			errMsg := fmt.Sprintf("[%s] panic file:[%s:%v] recovered:\n%s\n%s", "gmock", file, line, err, string(debug.Stack()))
			Log.Error(errMsg)
		}
	}
}
