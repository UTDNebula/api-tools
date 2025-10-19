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

// SplitWriter routes writes to multiple underlying writers.
type SplitWriter struct {
	writers []io.Writer
}

// NewSplitWriter constructs a SplitWriter that fans out writes to the provided writers.
func NewSplitWriter(writers ...io.Writer) *SplitWriter {
	return &SplitWriter{writers: writers}
}

// Write copies the provided bytes to each underlying writer.
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

// Lverbose enables verbose logging on Logger instances and global loggers.
const Lverbose = 1 << 7

// Logger extends log.Logger with helper methods that respect the verbose flag.
type Logger struct {
	log.Logger
}

// NewLogger constructs a Logger that writes to out with the given prefix and flags.
func NewLogger(out io.Writer, prefix string, flag int) *Logger {
	return &Logger{*log.New(out, prefix, flag)}
}

// VPrintf prints using fmt.Printf semantics when the verbose flag is set.
func (logger *Logger) VPrintf(format string, vars ...any) {
	flags := logger.Flags()
	if flags&Lverbose != 0 {
		logger.Printf(format, vars...)
	}
}

// VPrint prints text when the verbose flag is set.
func (logger *Logger) VPrint(text string) {
	flags := logger.Flags()
	if flags&Lverbose != 0 {
		logger.Print(text)
	}
}

// VPrintln prints text with a newline when the verbose flag is set.
func (logger *Logger) VPrintln(text string) {
	flags := logger.Flags()
	if flags&Lverbose != 0 {
		logger.Println(text)
	}
}

// VPrintf prints through the package-level logger when the verbose flag is set.
func VPrintf(format string, vars ...any) {
	flags := log.Flags()
	if flags&Lverbose != 0 {
		log.Printf(format, vars...)
	}
}

// VPrint prints text through the package-level logger when the verbose flag is set.
func VPrint(text string) {
	flags := log.Flags()
	if flags&Lverbose != 0 {
		log.Print(text)
	}
}

// VPrintln prints text with a newline through the package-level logger when the verbose flag is set.
func VPrintln(text string) {
	flags := log.Flags()
	if flags&Lverbose != 0 {
		log.Println(text)
	}
}
