# mqtt-match [![GoDoc](https://godoc.org/github.com/farshidtz/mqtt-match?status.svg)](https://godoc.org/github.com/farshidtz/mqtt-match) [![Build Status](https://travis-ci.org/farshidtz/mqtt-match.svg?branch=master)](https://travis-ci.org/farshidtz/mqtt-match)

Match mqtt formatted topic strings to strings, e.g. `foo/+` should match `foo/bar`.

### Usage

```go
package main

import (
    "fmt"
    mqtttopic "github.com/farshidtz/mqtt-match"
)

func main(){
    fmt.Println(mqtttopic.Match("foo/+", "foo/bar"))
    // true
}
```
### Copyrights Notice
This package is a translation of [mqtt-match](https://github.com/ralphtheninja/mqtt-match) for Go.
