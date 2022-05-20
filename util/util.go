package util

import (
	"fmt"
	"log"
	"os"
)

type logger struct {
	tag string
	log *log.Logger
}
func NewLogger(tag string) *logger {
	return &logger{tag: tag, log: log.New(os.Stdout, fmt.Sprintf("[%v] ", tag), log.LstdFlags)}
}
func (l *logger) SetTag(tag string) {
	l.tag=tag
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
