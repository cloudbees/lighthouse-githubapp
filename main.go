package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cloudbees/lighthouse-githubapp/pkg/loghelpers"

	"github.com/cloudbees/lighthouse-githubapp/pkg/flags"
	"github.com/cloudbees/lighthouse-githubapp/pkg/hook"
	"github.com/sirupsen/logrus"
	muxtrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/gorilla/mux"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

func main() {
	loghelpers.InitLogrus()

	if flags.DebugLogging.Value() {
		logrus.SetLevel(logrus.DebugLevel)
	}

	logrus.Info("Lighthouse GitHub App is starting")

	if flags.DataDogEnabled.Value() {
		tracer.Start()
		defer tracer.Stop()
	}

	router := muxtrace.NewRouter(muxtrace.WithServiceName("lighthouse-githubapp"))

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
		err = server.Shutdown(ctx)
		if err != nil {
			logrus.Errorf("unable to shutdown cleanly: %s", err)
		}
	}()

	err = server.ListenAndServe()
	logrus.Fatalf(err.Error())
}
