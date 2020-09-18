package logger

import (
	"fmt"
	"os"
	"runtime"

	log "github.com/jeanphorn/log4go"
)

// Logger application logger
type Logger struct {
	logger log.Logger
}

// NewLogger constructs a logger object
func NewLogger() *Logger {

	logger := log.NewDefaultLogger(log.DEBUG)

	return &Logger{
		logger: logger,
	}
}

// Info log information
func Info(arg0 interface{}, args ...interface{}) {
	l := log.NewDefaultLogger(log.DEBUG)
	l.Log(log.INFO, getSource(), fmt.Sprintf(arg0.(string), args...))
}

// Debug log debug
func Debug(arg0 interface{}, args ...interface{}) {
	l := log.NewDefaultLogger(log.DEBUG)
	l.Log(log.DEBUG, getSource(), fmt.Sprintf(arg0.(string), args...))
}

// Warning log warnings
func Warning(arg0 interface{}, args ...interface{}) {
	l := log.NewDefaultLogger(log.DEBUG)
	l.Log(log.WARNING, getSource(), fmt.Sprintf(arg0.(string), args...))
}

// Error log errors
func Error(arg0 interface{}, args ...interface{}) {
	l := log.NewDefaultLogger(log.DEBUG)
	l.Log(log.ERROR, getSource(), fmt.Sprintf(arg0.(string), args...))
}

// Fatal log fatal errors
func Fatal(arg0 interface{}, args ...interface{}) {
	l := log.NewDefaultLogger(log.DEBUG)
	l.Log(log.CRITICAL, getSource(), fmt.Sprintf(arg0.(string), args...))
	l.Close()
	os.Exit(1)
}

func getSource() (source string) {
	if pc, _, line, ok := runtime.Caller(2); ok {
		source = fmt.Sprintf("%s:%d", runtime.FuncForPC(pc).Name(), line)
	}
	return
}
