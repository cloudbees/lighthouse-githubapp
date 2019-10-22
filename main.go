package main

import (
	"net/http"
	"os"

	"github.com/cloudbees/lighthouse-githubapp/pkg/hook"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetFormatter(CreateDefaultFormatter())

	mux := http.NewServeMux()

	handler, err := hook.NewHook()
	if err != nil {
		logrus.WithError(err).Fatalf("failed to create hook")
	}

	handler.Handle(mux)

	logrus.Infof("Lighthouse GitHub App is now listening on path %s and port %s for WebHooks", handler.Path, handler.Port)

	err = http.ListenAndServe(":"+handler.Port, mux)
	logrus.Fatalf(err.Error())
}

// CreateDefaultFormatter creates a default JSON formatter
func CreateDefaultFormatter() logrus.Formatter {
	if os.Getenv("LOGRUS_FORMAT") == "text" {
		return &logrus.TextFormatter{
			ForceColors:      true,
			DisableTimestamp: true,
		}
	}
	jsonFormat := &logrus.JSONFormatter{}
	if os.Getenv("LOGRUS_JSON_PRETTY") == "true" {
		jsonFormat.PrettyPrint = true
	}
	return jsonFormat
}
