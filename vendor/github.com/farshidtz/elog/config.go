package elog

import (
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

const (
	NoTrace   = 1              // no file name or line lumber
	LongFile  = log.Llongfile  // absolute file path and line number: /home/user/go/main.go:15
	ShortFile = log.Lshortfile // file name and line number: main.go:15
)

type Config struct {
	// Writer is the destination to which log data is written
	// Default: os.Stdout
	Writer       io.Writer
	// TimeFormat is the format of time in the prefix (compatible with https://golang.org/pkg/time/#pkg-constants)
	// Default: "2006/01/02 15:04:05"
	TimeFormat   string
	// Trace is to control the tracing information
	// Default: NoTrace
	Trace        int
	// DebugEnabled can be used to pass a parsed command-line boolean flag to enable the debugging
	// Default: nil
	DebugEnabled *bool
	// DebugEnvVar is the environment variable used to enable debugging when set to 1
	// Default: "DEBUG"
	DebugEnvVar  string
	// DebugPrefix is the prefix used when logging with Debug methods and Errorf
	// Default: "[debug] "
	DebugPrefix  string
	// DebugTrace is to control the tracing information when in debugging mode
	// Default: ShortFile
	DebugTrace   int
}

func initConfig(config *Config) *Config {
	var conf = &Config{
		// default configurations
		Writer:     os.Stdout,
		TimeFormat: "2006/01/02 15:04:05",
		Trace:      NoTrace,
		// debugging config
		DebugEnvVar:  "DEBUG",
		DebugEnabled: nil,
		DebugPrefix:  "[debug] ",
		DebugTrace:   ShortFile,
	}

	if config != nil {
		if config.Writer != nil {
			conf.Writer = config.Writer
		}
		if config.TimeFormat != "" {
			conf.TimeFormat = config.TimeFormat
		}
		if config.Trace == NoTrace || config.Trace == 0 {
			conf.Trace = 0
		} else {
			conf.Trace = config.Trace
		}
		// debugging conf
		if config.DebugEnabled != nil {
			conf.DebugEnabled = config.DebugEnabled
		}
		if config.DebugEnvVar != "" {
			conf.DebugEnvVar = config.DebugEnvVar
		}
		if config.DebugPrefix != "" {
			conf.DebugPrefix = config.DebugPrefix
		}
		if config.DebugTrace == NoTrace {
			conf.DebugTrace = 0
		} else if config.DebugTrace == 0 {
			conf.DebugTrace = ShortFile // default
		} else {
			conf.DebugTrace = config.DebugTrace

		}
	}

	conf.TimeFormat = fmt.Sprintf("%s ", strings.TrimSpace(conf.TimeFormat))
	if conf.DebugEnabled == nil {
		var debug bool
		// Enable debugging if environment variable is set
		v, err := strconv.Atoi(os.Getenv(conf.DebugEnvVar))
		if err == nil && v == 1 {
			debug = true
		}
		conf.DebugEnabled = &debug
	}

	return conf
}
