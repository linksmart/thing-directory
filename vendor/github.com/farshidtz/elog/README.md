# elog
[![GoDoc](https://godoc.org/github.com/farshidtz/elog?status.svg)](https://godoc.org/github.com/farshidtz/elog)
[![Build Status](https://travis-ci.org/farshidtz/elog.svg?branch=master)](https://travis-ci.org/farshidtz/elog)

elog extends Go's built-in [log](https://golang.org/pkg/log) package to enable simple levelled logging and to modify the formatting. 

## Debugging mode
The debugging mode can be enabled by [setting the configured environment variable](https://github.com/farshidtz/elog#setting-an-environment-variable) ("DEBUG" by default) to 1. Alternatively, the debugging can be enabled using a [flag](https://github.com/farshidtz/elog#enable-debugging-with-a-flag).

## Usage
Get the package

    go get github.com/farshidtz/elog
Import
```go
import "github.com/farshidtz/elog"
```

Examples
```go
10	// Initialize with the default configuration
11	logger := elog.New("[main] ", nil)
12	
13	logger.Println("Hello world!")
14	logger.Debugln("Hello world!")
15	logger.Fatalln("Hello world!")
```
Debugging not enabled
```
2016/07/14 16:47:04 [main] Hello world!
2016/07/14 16:47:04 [main] Hello world!
exit with status 1
```
Debugging enabled
```
2016/07/14 16:47:04 [main] main.go:13: Hello world!
2016/07/14 16:47:04 [debug] main.go:14 Hello world!
2016/07/14 16:47:04 [main] main.go:15 Hello world!
exit with status 1
```
### Error logging
```go
01	package main
02
03	import "github.com/farshidtz/elog"
04
05	var logger *elog.Logger
06
07	func main() {
08		logger = elog.New("[main] ", nil)
09
10		v, err := divide(10, 0)
11		if err != nil {
12			logger.Fatalln(err)
13		}
14		logger.Println(v)
15	}
16
17	func divide(a, b int) (float64, error) {
18		if b == 0 {
19			return 0, logger.Errorf("Cannot divide by zero")
20			// The error is logged in debugging mode
21		}
22		return float64(a) / float64(b), nil
23	}
```
Debugging not enabled
```
2016/07/20 16:38:31 [main] main.go:12: Cannot divide by zero
```
Debugging enabled
```
2016/07/20 16:38:31 [debug] main.go:19: Cannot divide by zero
2016/07/20 16:38:31 [main] main.go:12: Cannot divide by zero
```
### Configuration
```go
10	logger := elog.New("[I] ", &elog.Config{
11	  TimeFormat: time.RFC3339, 
12	  DebugPrefix: "[D] ", 
13	})
14	
15	logger.Println("Hello world!")
16	logger.Debugln("Hello world!")
17	logger.Fatalln("Hello world!")
```
Debugging not enabled
```
2016-07-14T16:57:15Z [I] Hello world!
2016-07-14T16:57:15Z [I] Hello world!
exit with status 1
```
Debugging enabled
```
2016-07-14T16:57:15Z [I] main.go:15 Hello world!
2016-07-14T16:57:15Z [D] main.go:16 Hello world!
2016-07-14T16:57:15Z [I] main.go:17 Hello world!
exit with status 1
```
### Initialization with init()
Alternatively, a global logger can be instantiated in the init() function and used throughout the package.  
```go
import "github.com/farshidtz/elog"

var logger *elog.Logger

func init() {
	logger = elog.New("[main] ", nil)
}
```

### Enable debugging with a flag
```go
package main

import (
	"github.com/farshidtz/elog"
	"flag"
)

var debugFlag = flag.Bool("d", false, "Enable debugging")
func main() {
	flag.Parse()
	logger := elog.New("[main] ", &elog.Config{
	  DebugEnabled: debugFlag,
	})
	
	logger.Println("Hello World!")
	logger.Debugln("Hello World!")
}
```
Example (Windows PowerShell):
```
PS C:\logging> .\main.exe
2016/07/20 15:55:31 [main] Hello World!

PS C:\logging> .\main.exe -d
2016/07/20 15:55:32 [main] main.go:15: Hello World!
2016/07/20 15:55:32 [debug] main.go:16: Hello World!
```

### Setting an environment variable
```
Unix:
export DEBUG=1

Command Prompt:
set DEBUG=1

PowerShell:
$env:DEBUG=1
```
## Documentation
For Go log's build-in methods: [golang.org/pkg/log](https://golang.org/pkg/log)

For extended elog methods: [godoc.org/github.com/farshidtz/elog](https://godoc.org/github.com/farshidtz/elog)