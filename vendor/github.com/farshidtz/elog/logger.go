package elog

import (
	"fmt"
	"log"
)

type Logger struct {
	*log.Logger
	debug *log.Logger
}

/*
New creates a new Logger.
The prefix variable is the prefix used when logging with Print, Fatal, and Panic methods.
The config variable is a struct with optional fields.

When config is set to nil, the default configuration will be used.
*/
func New(prefix string, config *Config) *Logger {
	conf := initConfig(config)

	var logger Logger
	if *conf.DebugEnabled {
		logger.Logger = log.New(&writer{conf.Writer, conf.TimeFormat}, prefix, conf.DebugTrace)
		logger.debug = log.New(&writer{conf.Writer, conf.TimeFormat}, conf.DebugPrefix, conf.DebugTrace)
	} else {
		logger.Logger = log.New(&writer{conf.Writer, conf.TimeFormat}, prefix, conf.Trace)
	}
	return &logger
}

// Debug has similar arguments to those of fmt.Print. It prints to the logger only when debugging is enabled.
func (l *Logger) Debug(a ...interface{}) {
	if l.debug != nil {
		l.debug.Output(2, fmt.Sprint(a...))
	}
}

// Debugf has similar arguments to those of fmt.Printf. It prints to the logger only when debugging is enabled.
func (l *Logger) Debugf(format string, a ...interface{}) {
	if l.debug != nil {
		l.debug.Output(2, fmt.Sprintf(format, a...))
	}
}

// Debugln has similar arguments to those of fmt.Println. It prints to the logger only when debugging is enabled.
func (l *Logger) Debugln(a ...interface{}) {
	if l.debug != nil {
		l.debug.Output(2, fmt.Sprintln(a...))
	}
}

/*
DebugOutput is similar to Debug but allows control over the scope of tracing. (log.Output for debugging)

E.g. :
  [main.go]

  10  apiError("error message")
  11
  12  // A function to log formatted API errors
  13  func apiError(s string){
  14     logger.DebugOutput(2, "API Error:", s)
  15  }

  Prints: 2016/07/19 17:34:10 [debug] main.go:10: API Error: error message
*/
func (l *Logger) DebugOutput(calldepth int, s string) {
	if l.debug != nil {
		l.debug.Output(calldepth+1, s)
	}
}

/*
Errorf has similar arguments and signature to those of fmt.Errorf. In addition to formatting and returning error, Errorf prints the error message to the logger when debugging is enabled.

E.g. :
  [example.go]

  19  func DoSomething() error {
  20    err, v := divide(10, 0)
  21    if err != nil {
  22      return logger.Errorf("Division error: %s", err)
  23    }
  24
  25    return nil
  24  }

  Prints: 2016/07/19 17:44:22 [debug] example.go:22: Division error: cannot divide by zero
*/
func (l *Logger) Errorf(format string, a ...interface{}) error {
	if l.debug != nil {
		l.debug.Output(2, fmt.Sprintf(format, a...))
	}
	return fmt.Errorf(format, a...)
}
