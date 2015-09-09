package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

var (
	Trace   *log.Logger
	Log     *log.Logger
	Warning *log.Logger
	Err     *log.Logger
	Prefix  string
)

func InitLogger(
	traceHandle io.Writer,
	infoHandle io.Writer,
	warningHandle io.Writer,
	errorHandle io.Writer,
	prefix string) {

	Trace = log.New(traceHandle, fmt.Sprintf("[%s] TRACE: ", prefix), 0)
	Log = log.New(infoHandle, fmt.Sprintf("[%s] ", prefix), 0)
	Warning = log.New(warningHandle, fmt.Sprintf("[%s] WARNING: ", prefix), 0)
	Err = log.New(errorHandle, fmt.Sprintf("[%s] ERROR: ", prefix), 0)
	Prefix = prefix
}

// Formats error
func Error(err error) error {
	return fmt.Errorf("%s Error: %s", Prefix, err.Error())
}

// Formats message into error
func Errorf(message string) error {
	return fmt.Errorf("%s Error: %s", Prefix, message)
}

// Writes formatted error to HTTP ResponseWriter
func HTTPErrorResponse(code int, msg string, w http.ResponseWriter) {
	e := map[string]interface{}{
		"code":    code,
		"message": msg,
	}
	b, _ := json.Marshal(&e)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(b)
}

//func main() {
//	Init(ioutil.Discard, os.Stdout, os.Stdout, os.Stderr)

//	Trace.Println("I have something standard to say")
//	Info.Println("Special Information")
//	Warning.Println("There is something you need to know about")
//	Error.Println("Something has failed")
//}
