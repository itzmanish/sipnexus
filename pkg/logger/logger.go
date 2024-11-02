package logger

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Logger interface {
	Info(msg string)
	Infof(format string, v ...interface{})
	Warn(msg string)
	Warnf(format string, v ...interface{})
	Error(msg string)
	Errorf(format string, v ...interface{})
	Fatal(msg string)
	Fatalf(format string, v ...interface{})
}

type ZeroLogger struct {
	log zerolog.Logger
}

func init() {
	zerolog.CallerMarshalFunc = func(pc uintptr, file string, line int) string {
		return filepath.Base(file) + ":" + strconv.Itoa(line)
	}
	log.Logger = log.Logger.Output(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Caller().Logger()
}

func NewLogger() Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	return &ZeroLogger{
		log: log.Logger,
	}
}

func (l *ZeroLogger) InternalLogger() zerolog.Logger {
	return l.log
}

func (l *ZeroLogger) Info(msg string) {
	l.log.Info().Msg(msg)
}

func (l *ZeroLogger) Infof(format string, v ...interface{}) {
	l.log.Info().Msgf(format, v...)
}

func (l *ZeroLogger) Warn(msg string) {
	l.log.Warn().Msg(msg)
}

func (l *ZeroLogger) Warnf(format string, v ...interface{}) {
	l.log.Warn().Msgf(format, v...)
}

func (l *ZeroLogger) Error(msg string) {
	l.log.Error().Msg(msg)
}

func (l *ZeroLogger) Errorf(format string, v ...interface{}) {
	l.log.Error().Msgf(format, v...)
}

func (l *ZeroLogger) Fatal(msg string) {
	l.log.Fatal().Msg(msg)
}

func (l *ZeroLogger) Fatalf(format string, v ...interface{}) {
	l.log.Fatal().Msgf(format, v...)
}
