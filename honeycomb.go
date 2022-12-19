package app

import (
	"fmt"
	"os"

	_ "github.com/honeycombio/honeycomb-opentelemetry-go"
	"github.com/honeycombio/opentelemetry-go-contrib/launcher"
)

func ConfigureOpenTelemetry() (cleanup func(), err error) {
	cleanup = func() {}
	otelShutdown, err := launcher.ConfigureOpenTelemetry(
		launcher.WithResourceAttributes(map[string]string{
			"fly.region":   os.Getenv("FLY_REGION"),
			"fly.alloc_id": os.Getenv("FLY_ALLOC_ID"),
			"fly.app_name": os.Getenv("FLY_APP_NAME"),
		}))
	if err != nil {
		err = fmt.Errorf("setting up OTel SDK: %w", err)
	} else {
		cleanup = otelShutdown
	}
	return
}
