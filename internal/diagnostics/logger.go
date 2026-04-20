package diagnostics

import (
	"log"
	"os"
)

func NewLogger() *log.Logger {
	return log.New(redactingWriter{out: os.Stderr}, "", log.LstdFlags)
}
