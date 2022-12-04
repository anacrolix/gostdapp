package app

import (
	"os"

	"github.com/anacrolix/backtrace"
	"github.com/anacrolix/envpprof"
	"github.com/anacrolix/log"
)

// Doesn't return on error.
func Run(mainErr func() error) {
	err := mainErr()
	envpprof.Stop()
	if err != nil {
		log.Printf("error in main: %v%s", err, backtrace.Sprint(err))
		os.Exit(1)
	}
}
