package log

import (
	"log"
	"os"
)

var (
	Error *log.Logger
	Info  *log.Logger
	Debug *log.Logger
)

func init() {
	Error = log.New(os.Stdout, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	Info = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	Debug = log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)

}
