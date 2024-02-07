/*
	This file contains a log.Logger wrapper that provides some "verbose-only" variants of built-in logging functions.
	These "verbose-only" functions, all of which start with 'V', will only print if the custom 'verbose' flag is specified in
	the log.Logger being used.

	Additionally, a "SplitWriter" implementation of io.Writer is provided which supports writing to
*/

package utils

import (
	"io"
	"log"
)

// Custom io.Writer for routing writing to multiple sub-writers
type SplitWriter struct {
	writers []io.Writer
}

// Constructor for utils.SplitWriter
func NewSplitWriter(writers ...io.Writer) *SplitWriter {
	return &SplitWriter{writers: writers}
}

// Writes the specified bytes to every sub-writer of the SplitWriter
func (splitWriter *SplitWriter) Write(p []byte) (n int, err error) {
	type writeResult struct {
		n   int
		err error
	}
	// Perform synchronous write across writers with result channel
	c := make(chan writeResult)
	for _, w := range splitWriter.writers {
		go func(writer io.Writer) {
			n, err := writer.Write(p)
			c <- writeResult{n, err}
		}(w)
	}
	// Wait for all results from channel, reports total bytes written and immediately returns on error
	for range splitWriter.writers {
		res := <-c
		if res.err == nil {
			n += res.n
		} else {
			break
		}
	}
	return n, err
}

// Verbose logging flag, only works with the utils.Logger verbose functions
const Lverbose = 1 << 7

// Extension of log.Logger that supports a verbose logging flag; verbose printing functions start with 'V'
type Logger struct {
	log.Logger
}

func NewLogger(out io.Writer, prefix string, flag int) *Logger {
	return &Logger{*log.New(out, prefix, flag)}
}

// Verbose-only variant of Logger.Printf
func (logger *Logger) VPrintf(format string, vars ...any) {
	flags := logger.Flags()
	if flags&Lverbose != 0 {
		logger.Printf(format, vars...)
	}
}

// Verbose-only variant of Logger.Print
func (logger *Logger) VPrint(text string) {
	flags := logger.Flags()
	if flags&Lverbose != 0 {
		logger.Print(text)
	}
}

// Verbose-only variant of Logger.Println
func (logger *Logger) VPrintln(text string) {
	flags := logger.Flags()
	if flags&Lverbose != 0 {
		logger.Println(text)
	}
}

// Verbose-only variant of log.Printf
func VPrintf(format string, vars ...any) {
	flags := log.Flags()
	if flags&Lverbose != 0 {
		log.Printf(format, vars...)
	}
}

// Verbose-only variant of log.Print
func VPrint(text string) {
	flags := log.Flags()
	if flags&Lverbose != 0 {
		log.Print(text)
	}
}

// Verbose-only variant of log.Println
func VPrintln(text string) {
	flags := log.Flags()
	if flags&Lverbose != 0 {
		log.Println(text)
	}
}
