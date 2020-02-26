package hook

import (
	"fmt"
	"github.com/cenkalti/backoff"
	"github.com/cloudbees/lighthouse-githubapp/pkg/util"
	"net/http"
	"time"

	"github.com/cloudbees/lighthouse-githubapp/pkg/flags"
	"github.com/jenkins-x/go-scm/scm"
)

func (o *HookOptions) handleWebHookRequests(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		// liveness probe etc
		util.TraceLogger(r.Context()).WithField("method", r.Method).Debug("invalid http method so returning index")
		o.getIndex(w, r)
		return
	}
	util.TraceLogger(r.Context()).Debug("about to parse webhook")

	scmClient, _, _, err := o.createSCMClient("")
	if err != nil {
		util.TraceLogger(r.Context()).Errorf("failed to create SCM scmClient: %s", err.Error())
		responseHTTPError(w, http.StatusInternalServerError, fmt.Sprintf("500 Internal Server Error: Failed to parse webhook: %s", err.Error()))
		return
	}

	secretFn := func(webhook scm.Webhook) (string, error) {
		return flags.HmacToken.Value(), nil
	}

	webhook, err := scmClient.Webhooks.Parse(r, secretFn)
	if err != nil {
		util.TraceLogger(r.Context()).Warnf("failed to parse webhook: %s", err.Error())

		responseHTTPError(w, http.StatusInternalServerError, fmt.Sprintf("500 Internal Server Error: Failed to parse webhook: %s", err.Error()))
		return
	}
	if webhook == nil {
		util.TraceLogger(r.Context()).Error("no webhook was parsed")

		responseHTTPError(w, http.StatusInternalServerError, "500 Internal Server Error: No webhook could be parsed")
		return
	}

	repository := webhook.Repository()
	l := util.TraceLogger(r.Context()).WithFields(map[string]interface{}{
		"FullName": repository.FullName,
		"Webhook":  webhook.Kind(),
	})
	installHook, ok := webhook.(*scm.InstallationHook)
	if ok {
		if installHook.Installation.ID == 0 {
			responseHTTPError(w, http.StatusInternalServerError, "500 Internal Server Error: missing installation ID")
			return
		}
		l = l.WithField("Installation", installHook.Installation.ID)
		l.Info("invoking Installation handler")
		err = o.onInstallHook(r.Context(), l, installHook)
		if err != nil {
			responseHTTPError(w, http.StatusInternalServerError, "500 Internal Server Error: %s", err.Error())
		} else {
			writeResult(l, w, "OK")
		}
		return
	}

	installRef := webhook.GetInstallationRef()
	if installRef == nil || installRef.ID == 0 {
		l.WithField("Hook", webhook).Error("no installation reference was passed for webhook")

		responseHTTPError(w, http.StatusInternalServerError, "500 Internal Server Error: No installation in webhook")
		return
	}

	err = retry(time.Second*30, func() error {
		return o.onGeneralHook(r.Context(), l, installRef, webhook)
	})

	if err != nil {
		l.WithError(err).Error("failed to process webhook after 30 seconds")
		responseHTTPError(w, http.StatusInternalServerError, "500 Internal Server Error: %s", err.Error())
	}
	writeResult(l, w, "OK")
	return
}

func retry(maxElapsedTime time.Duration, f func() error) error {
    bo := backoff.NewExponentialBackOff()
    bo.MaxElapsedTime = maxElapsedTime
    bo.Reset()
    return backoff.Retry(f, bo)
}
