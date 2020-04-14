package hook

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/cloudbees/lighthouse-githubapp/pkg/util"

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

	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		util.TraceLogger(r.Context()).Errorf("failed to Read Body: %s", err.Error())
		responseHTTPError(w, http.StatusInternalServerError, fmt.Sprintf("500 Internal Server Error: Read Body: %s", err.Error()))
		return
	}

	err = r.Body.Close() // must close
	if err != nil {
		util.TraceLogger(r.Context()).Errorf("failed to Close Body: %s", err.Error())
		responseHTTPError(w, http.StatusInternalServerError, fmt.Sprintf("500 Internal Server Error: Read Close: %s", err.Error()))
		return
	}

	r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

	webhook, err := scmClient.Webhooks.Parse(r, o.secretFn)
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

	l.Debugf("got hook %s", webhook.Kind())
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

	installRepositoryHook, ok := webhook.(*scm.InstallationRepositoryHook)
	if ok {
		if installRepositoryHook.Installation.ID == 0 {
			responseHTTPError(w, http.StatusInternalServerError, "500 Internal Server Error: missing installation ID")
			return
		}
		l = l.WithField("Installation", installRepositoryHook.Installation.ID)
		l.Info("invoking Installation Repository handler")
		err = o.onInstallRepositoryHook(r.Context(), l, installRepositoryHook)
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

	githubDeliveryEvent := r.Header.Get("X-GitHub-Delivery")
	githubEventType := r.Header.Get("X-GitHub-Event")
	err = o.onGeneralHook(r.Context(), l, installRef, webhook, githubEventType, githubDeliveryEvent, bodyBytes)

	if err != nil {
		l.WithError(err).Errorf("failed to process webhook for '%s'", repository.FullName)
		responseHTTPError(w, http.StatusInternalServerError, "500 Internal Server Error: %s", err.Error())
	}
	writeResult(l, w, "OK")
	return
}
