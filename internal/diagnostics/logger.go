package diagnostics

import (
	"io"
	"log"
)

func NewLogger() *log.Logger {
	return log.New(io.Discard, "", log.LstdFlags)
}
