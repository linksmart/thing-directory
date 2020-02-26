package elog

import (
	"io"
	"time"
)

type writer struct {
	io.Writer
	timeFormat string
}

func (w writer) Write(b []byte) (n int, err error) {
	return w.Writer.Write(append([]byte(time.Now().Format(w.timeFormat)), b...))
}

/*
NewWriter returns a new io.Writer that writes timestamps as prefix

E.g. usage:

	logger := log.New(elog.NewWriter(os.Stdout), "[info] ", 0)
	logger.Println("Hello.")

	Prints: 2016/07/19 17:34:10 [info] Hello.
*/
func NewWriter(w io.Writer, timeFormat ...string) *writer {
	tf := "2006-01-02 15:04:05 "
	if len(timeFormat) == 1 {
		tf = timeFormat[0]
	}
	return &writer{w, tf}
}
