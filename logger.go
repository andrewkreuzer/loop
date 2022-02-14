package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
)

// Logger ...
var Logger *Log

// Log ...
type Log struct {
	logBuffer     bytes.Buffer
	logFile       *os.File
	writeToStdout bool

	InfoLogger  *log.Logger
	WarnLogger  *log.Logger
	ErrorLogger *log.Logger
	FatalLogger *log.Logger
}

// NewLogger ...
func NewLogger(filename string, writeToStdout bool) *Log {
	Logger = new(Log)

	Logger.writeToStdout = writeToStdout
	Logger.InfoLogger = log.New(&Logger.logBuffer, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	Logger.WarnLogger = log.New(&Logger.logBuffer, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
	Logger.ErrorLogger = log.New(&Logger.logBuffer, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	Logger.FatalLogger = log.New(&Logger.logBuffer, "FATAL: ", log.Ldate|log.Ltime|log.Lshortfile)

	var err error
	Logger.logFile, err = os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		Logger.Fatal("Unable to open log file: ", filename)
	}

	return Logger
}

func getLogger() *Log {
	return Logger
}

func (log *Log) writeToFile() {
	_, err := fmt.Fprint(log.logFile, &log.logBuffer)
	if err != nil {
		fmt.Println(err)
	}
}

// Warn ...
func (log *Log) Warn(message ...interface{}) {
	log.WarnLogger.Println(message...)

	log.writeToFile()
	if log.writeToStdout {
		log.logBuffer.WriteTo(os.Stdout)
	}
}

// Info ...
func (log *Log) Info(message ...interface{}) {
	log.InfoLogger.Println(message...)

	log.writeToFile()
	if log.writeToStdout {
		log.logBuffer.WriteTo(os.Stdout)
	}
}

// Error ...
func (log *Log) Error(message ...interface{}) {
	log.ErrorLogger.Println(message...)

	log.writeToFile()
	if log.writeToStdout {
		log.logBuffer.WriteTo(os.Stdout)
	}
}

// Fatal ...
func (log *Log) Fatal(message ...interface{}) {
	log.FatalLogger.Println(message...)

	log.writeToFile()
	if log.writeToStdout {
		log.logBuffer.WriteTo(os.Stdout)
	}

	os.Exit(1)
}
