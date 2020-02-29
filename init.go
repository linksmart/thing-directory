// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package main

import (
	"log"
	"os"
)

const (
	EnvVerbose        = "VERBOSE"          // print extra information e.g. line number)
	EnvDisableLogTime = "DISABLE_LOG_TIME" // disable timestamp in logs
)

func init() {
	log.SetOutput(os.Stdout)
	log.SetFlags(0)

	logFlags := log.LstdFlags
	if evalEnv(EnvDisableLogTime) {
		logFlags = 0
	}
	if evalEnv(EnvVerbose) {
		logFlags = logFlags | log.Lshortfile
	}
	log.SetFlags(logFlags)
}

// evalEnv returns the boolean value of the env variable with the given key
func evalEnv(key string) bool {
	return os.Getenv(key) == "1" || os.Getenv(key) == "true" || os.Getenv(key) == "TRUE"
}
