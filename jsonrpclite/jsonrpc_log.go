package jsonrpclite

import (
	"fmt"
	"sync"
	"time"
)

type LogLevel uint8

const (
	DebugLevel = iota
	TraceLevel
	InfoLevel
	WarningLevel
	ErrorLevel
)

var (
	levels = map[LogLevel]string{
		DebugLevel:   "Debug",
		TraceLevel:   "Trace",
		InfoLevel:    "Info",
		WarningLevel: "Warning",
		ErrorLevel:   "Error",
	}
)

type RpcLogger interface {
	Debug(msg string)   //Write the debug log.
	Trace(msg string)   //Write the trace log.
	Info(msg string)    //Write the info log.
	Warning(msg string) //Write the warning log.
	Error(msg string)   //Write the error log.
}

var logger = newConsoleLogger()

//SetRpcLogger call this method to register custom logger into the JsonRpcLite
func SetRpcLogger(l RpcLogger) {
	logger = l
}

//Default logger for print log on console.
type rpcConsoleLogger struct {
	locker *sync.Mutex
}

//Create the default console logger.
func newConsoleLogger() RpcLogger {
	l := new(rpcConsoleLogger)
	l.locker = new(sync.Mutex)
	return l
}

//Write the log by different level.
func (l *rpcConsoleLogger) log(level LogLevel, msg string) {
	defer func() {
		var p = any(recover())
		if p != nil {
			l.locker.Unlock()
		}
	}()
	now := time.Now()
	timeStr := now.Format("2006-01-02 15:04:05.000000000")
	l.locker.Lock()
	fmt.Println(levels[level] + "- [" + timeStr + "] " + msg)
	l.locker.Unlock()
}

// Debug Write the debug log.
func (l *rpcConsoleLogger) Debug(msg string) {
	l.log(DebugLevel, msg)
}

// Trace Write the trace log.
func (l *rpcConsoleLogger) Trace(msg string) {
	l.log(TraceLevel, msg)
}

// Info Write the info log.
func (l *rpcConsoleLogger) Info(msg string) {
	l.log(InfoLevel, msg)
}

// Warning Write the warning log.
func (l *rpcConsoleLogger) Warning(msg string) {
	l.log(WarningLevel, msg)
}

//Write the error log.
func (l *rpcConsoleLogger) Error(msg string) {
	l.log(ErrorLevel, msg)
}
