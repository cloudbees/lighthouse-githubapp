package main

import (
	"context"
	"github.com/TV4/logrus-stackdriver-formatter"
	"github.com/cloudbees/lighthouse-githubapp/pkg/flags"
	"github.com/cloudbees/lighthouse-githubapp/pkg/hook"
	"github.com/cloudbees/lighthouse-githubapp/pkg/version"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	logrus.SetFormatter(stackdriver.NewFormatter(
		stackdriver.WithService("lighthouse-githubapp"),
		stackdriver.WithVersion(*version.GetBuildVersion()),
	))
	if flags.DebugLogging.Value() {
		logrus.SetLevel(logrus.DebugLevel)
	}

	logrus.Info("Lighthouse GitHub App is starting")

	router := mux.NewRouter()

	handler, err := hook.NewHook()
	if err != nil {
		logrus.WithError(err).Fatalf("failed to create hook")
	}

	handler.Handle(router)

	logrus.Infof("Lighthouse GitHub App is now listening on path %s and port %s for WebHooks", handler.Path, handler.Port)
	http.Handle("/", router)
	server := &http.Server{Addr: ":" + handler.Port, Handler: router}

	// Shutdown gracefully on SIGTERM or SIGINT
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sig
		logrus.Info("lighthouse github app is shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()
		server.Shutdown(ctx)
	}()

	err = server.ListenAndServe()
	logrus.Fatalf(err.Error())
}
