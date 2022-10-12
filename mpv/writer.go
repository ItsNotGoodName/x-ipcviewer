package mpv

import (
	"log"
	"strings"
)

type PrintWriter struct {
	name string
}

func NewPrintWriter(name string) *PrintWriter {
	return &PrintWriter{
		name: name,
	}
}

func (pw *PrintWriter) Write(p []byte) (n int, err error) {
	for _, s := range strings.Split(string(p), "\n") {
		if s == "" {
			continue
		}
		log.Printf("mpv: %s: %s", pw.name, s)
	}
	return len(p), nil
}
