package utils

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/juliotorresmoreno/doppler/config"
)

type Level string

const (
	INFO  Level = "INFO"
	WARN  Level = "WARN"
	ERROR Level = "ERROR"
)

var levelColors = map[Level]string{
	INFO:  "\033[32m", // verde
	WARN:  "\033[33m", // amarillo
	ERROR: "\033[31m", // rojo
}

const resetColor = "\033[0m"

type Logger interface {
	Register(url, method string, status int)
	RegisterWithLevel(level Level, url, method string, status int)
}

type LoggerBD struct {
	enabled bool
	logger  *log.Logger
}

func NewLogger() *LoggerBD {
	conf, _ := config.GetConfig()
	return &LoggerBD{
		enabled: conf.Logger,
		logger:  log.New(os.Stdout, "", log.LstdFlags),
	}
}

func (l *LoggerBD) Register(url, method string, status int) {
	l.RegisterWithLevel(INFO, url, method, status)
}

func (l *LoggerBD) RegisterWithLevel(level Level, url, method string, status int) {
	if !l.enabled {
		return
	}
	color, ok := levelColors[level]
	if !ok {
		color = ""
	}
	ts := time.Now().Format(time.RFC3339)
	l.logger.Printf("[%s%s%s] %s %s %d\n", color, level, resetColor, ts, fmt.Sprintf("%s %s", method, url), status)
}
