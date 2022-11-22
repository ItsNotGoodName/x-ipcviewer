package mpv

import (
	"log"
	"strings"
)

type LogWriter struct {
	name string
}

func NewLogWriter(name string) *LogWriter {
	return &LogWriter{
		name: name,
	}
}

func (lw *LogWriter) Write(p []byte) (n int, err error) {
	for _, s := range strings.Split(string(p), "\n") {
		if s == "" {
			continue
		}
		log.Printf("mpv: %s: %s", lw.name, s)
	}
	return len(p), nil
}
