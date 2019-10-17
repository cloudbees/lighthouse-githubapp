package main

import (
	"io/ioutil"
	"net/http"
	"os"

	"github.com/cloudbees/lighthouse-githubapp/pkg/flags"
	"github.com/cloudbees/lighthouse-githubapp/pkg/hook"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetFormatter(CreateDefaultFormatter())

	privateKeyFile := flags.AppPrivateKeyFile.Value()
	if privateKeyFile == "" {
		logrus.Fatalf("missing private key file environment variable $LHA_PRIVATE_KEY_FILE")
	}

	privateKey, err := ioutil.ReadFile(privateKeyFile)
	if err != nil {
		logrus.Fatalf("could not read private key file %s: %s", privateKeyFile, err)
	}
	if len(privateKey) == 0 {
		logrus.Fatalf("empty private key file %s", privateKeyFile, err)
	}

	mux := http.NewServeMux()

	handler := hook.NewHook(privateKeyFile)
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
