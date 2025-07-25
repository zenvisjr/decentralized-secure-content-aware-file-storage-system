package main

import (
	"fmt"
	"os"
	"sync"
	"time"
)

type AuditLogger struct {
	mu          sync.Mutex
	logFile     *os.File
	logFilePath string
}

// func NewAuditLogger(logFilePath, logFileName string) (*AuditLogger, error) {
// 	if err := os.MkdirAll(logFilePath, 0755); err != nil {
// 		return nil, err
// 	}
// 	fd, err := os.OpenFile(logFilePath+"/"+logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
// 	if err != nil {
// 		return nil, err
// 	}

// 	startLog := fmt.Sprintf("[%s] LOGGING STARTED....\n", time.Now().Format(time.RFC3339))
// 	fd.WriteString(startLog)
// 	fd.WriteString("\n")

// 	newAuditLogger := &AuditLogger{
// 		logFile:     fd,
// 		logFilePath: logFilePath,
// 	}

// 	return newAuditLogger, nil
// }

func simpleNewAuditLogger(logFileName string) (*AuditLogger, error) {
	fd, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	startLog := fmt.Sprintf("[%s] LOGGING STARTED....\n", time.Now().Format(time.RFC3339))
	fd.WriteString(startLog)
	fd.WriteString("\n")

	newAuditLogger := &AuditLogger{
		logFile:     fd,
		logFilePath: logFileName,
	}

	return newAuditLogger, nil
}


func (a *AuditLogger) Log(op, filekey, peer, status string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	logEntry := fmt.Sprintf("%s | %s | %s | %s | %s\n", time.Now().Format(time.RFC3339), op, filekey, peer, status)
	_, err := a.logFile.WriteString(logEntry)
	if err != nil {
		fmt.Println("Error writing to log file:", err)
	}
}

func (a *AuditLogger) Close() error {
	return a.logFile.Close()
}

func (a *AuditLogger) GetLogFilePath() string {
	return a.logFilePath
}

func (a *AuditLogger) GetLogFile() *os.File {
	return a.logFile
}
