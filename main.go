package main

import (
	"io/ioutil"
	"net/http"
	"os"

	"github.com/cloudbees/lighthouse-githubapp/flags"
	"github.com/sirupsen/logrus"
)

const (
	// HealthPath is the URL path for the HTTP endpoint that returns health status.
	HealthPath = "/health"
	// ReadyPath URL path for the HTTP endpoint that returns ready status.
	ReadyPath = "/ready"
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

	o := Options{
		Path:           "/hook",
		Port:           flags.HttpPort.Value(),
		Version:        "todo",
		PrivateKeyFile: privateKeyFile,
	}

	mux := http.NewServeMux()
	mux.Handle(HealthPath, http.HandlerFunc(o.health))
	mux.Handle(ReadyPath, http.HandlerFunc(o.ready))

	mux.Handle("/", http.HandlerFunc(o.defaultHandler))
	mux.Handle(o.Path, http.HandlerFunc(o.handleWebHookRequests))

	logrus.Infof("Lighthouse GitHub App is now listening on path %s and port %s for WebHooks", o.Path, o.Port)
	err = http.ListenAndServe(":"+o.Port, mux)
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

func responseHTTPError(w http.ResponseWriter, statusCode int, response string) {
	logrus.WithFields(logrus.Fields{
		"response":    response,
		"status-code": statusCode,
	}).Info(response)
	http.Error(w, response, statusCode)
}
