/*

Package elog extends Go's built-in log package to enable levelled logging with improved formatting.

Methods

Inherited from the log package (https://golang.org/pkg/log):
    func (l *Logger) Print(v ...interface{})
    func (l *Logger) Printf(format string, v ...interface{})
    func (l *Logger) Println(v ...interface{})
    func (l *Logger) Output(calldepth int, s string) error

    func (l *Logger) Fatal(v ...interface{})
    func (l *Logger) Fatalf(format string, v ...interface{})
    func (l *Logger) Fatalln(v ...interface{})

    func (l *Logger) Panic(v ...interface{})
    func (l *Logger) Panicf(format string, v ...interface{})
    func (l *Logger) Panicln(v ...interface{})

Extensions:
    func (l *Logger) Debug(a ...interface{})
    func (l *Logger) Debugf(format string, a ...interface{})
    func (l *Logger) Debugln(a ...interface{})
    func (l *Logger) DebugOutput(calldepth int, s string)

    func (l *Logger) Errorf(format string, a ...interface{}) error

*/
package elog