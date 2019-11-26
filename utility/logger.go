package utility

import (
	"fmt"
	"os"
	"runtime"

	log "github.com/jeanphorn/log4go"
)

// Logger application logger
type Logger struct {
	logger *log.Filter
}

// NewLogger constructs a logger object
func NewLogger(logSettingsPath, appName, folder string) *Logger {
	appDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Could not load log location >> %s", err)
	}

	_ = os.Mkdir(folder, os.ModePerm)

	log.LoadConfiguration(appDir + string(os.PathSeparator) + logSettingsPath)

	return &Logger{
		logger: log.LOGGER(appName),
	}
}

// Info log information
func (l *Logger) Info(arg0 interface{}, args ...interface{}) {
	l.logger.Log(log.INFO, getSource(), fmt.Sprintf(arg0.(string), args...))
}

// Debug log debug
func (l *Logger) Debug(arg0 interface{}, args ...interface{}) {
	l.logger.Log(log.DEBUG, getSource(), fmt.Sprintf(arg0.(string), args...))
}

// Warning log warnings
func (l *Logger) Warning(arg0 interface{}, args ...interface{}) {
	l.logger.Log(log.WARNING, getSource(), fmt.Sprintf(arg0.(string), args...))
}

// Error log errors
func (l *Logger) Error(arg0 interface{}, args ...interface{}) {
	l.logger.Log(log.ERROR, getSource(), fmt.Sprintf(arg0.(string), args...))
}

// Fatal log fatal errors
func (l *Logger) Fatal(arg0 interface{}, args ...interface{}) {
	l.logger.Log(log.CRITICAL, getSource(), fmt.Sprintf(arg0.(string), args...))
	l.logger.Close()
	os.Exit(1)
}

func getSource() (source string) {
	if pc, _, line, ok := runtime.Caller(2); ok {
		source = fmt.Sprintf("%s:%d", runtime.FuncForPC(pc).Name(), line)
	}
	return
}
