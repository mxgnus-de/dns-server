package logger

import (
	"os"

	"github.com/rs/zerolog"
)

var Logger zerolog.Logger

func Init() {
	logFile, err := os.OpenFile("log.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}

	out := zerolog.MultiLevelWriter(logFile, zerolog.ConsoleWriter{Out: os.Stdout})
	Logger = zerolog.New(out).With().Timestamp().Logger()
}

func CreateService(serviceName string) zerolog.Logger {
	return Logger.With().Str("service", serviceName).Logger()
}
