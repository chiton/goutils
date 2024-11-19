package log

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

// WriteToFile sets the output of logs to include writing to a file as well as writing to the default out.
// The file that is returned must be closed at the end of the process. Don't forget to do this later.
//    defer func(file *os.File) {
//	    _ = file.Close()
//    }(file)

type LogWriter struct {
}

const logLevel = "INF"

func WriteToFile(filePath string) *os.File {
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	wrt := io.MultiWriter(os.Stdout, file)
	log.SetOutput(wrt)

	return file
}

func (writer LogWriter) Write(bytes []byte) (int, error) {
	return fmt.Printf("[%s %s] %s", time.Now().UTC().Format("2006-01-02 15:04:05Z"), logLevel, string(bytes))
}
