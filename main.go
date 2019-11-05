package main

import (
	"github.com/TV4/logrus-stackdriver-formatter"
	"github.com/cloudbees/lighthouse-githubapp/pkg/hook"
	"github.com/cloudbees/lighthouse-githubapp/pkg/version"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"net/http"
)

func main() {
	logrus.SetFormatter(stackdriver.NewFormatter(
		stackdriver.WithService("lighthouse-githubapp"),
		stackdriver.WithVersion(*version.GetBuildVersion()),
	))

	router := mux.NewRouter()

	handler, err := hook.NewHook()
	if err != nil {
		logrus.WithError(err).Fatalf("failed to create hook")
	}

	handler.Handle(router)

	logrus.Infof("Lighthouse GitHub App is now listening on path %s and port %s for WebHooks", handler.Path, handler.Port)
	http.Handle("/", router)
	err = http.ListenAndServe(":"+handler.Port, router)
	logrus.Fatalf(err.Error())
}
