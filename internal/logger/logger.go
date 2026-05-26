package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

func New() zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339
	return zerolog.New(os.Stdout).With().Timestamp().Logger()
}
