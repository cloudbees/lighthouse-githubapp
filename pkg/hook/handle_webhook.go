package hook

import (
	"fmt"
	"net/http"

	"github.com/cloudbees/lighthouse-githubapp/pkg/flags"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/sirupsen/logrus"
)

func (o *HookOptions) handleWebHookRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		// liveness probe etc
		logrus.WithField("method", r.Method).Debug("invalid http method so returning index")
		o.getIndex(w, r)
		return
	}
	logrus.Debug("about to parse webhook")

	scmClient, _, _, err := o.createSCMClient("")
	if err != nil {
		logrus.Errorf("failed to create SCM scmClient: %s", err.Error())
		responseHTTPError(w, http.StatusInternalServerError, fmt.Sprintf("500 Internal Server Error: Failed to parse webhook: %s", err.Error()))
		return
	}

	secretFn := func(webhook scm.Webhook) (string, error) {
		return flags.HmacToken.Value(), nil
	}

	webhook, err := scmClient.Webhooks.Parse(r, secretFn)
	if err != nil {
		logrus.Errorf("failed to parse webhook: %s", err.Error())

		responseHTTPError(w, http.StatusInternalServerError, fmt.Sprintf("500 Internal Server Error: Failed to parse webhook: %s", err.Error()))
		return
	}
	if webhook == nil {
		logrus.Error("no webhook was parsed")

		responseHTTPError(w, http.StatusInternalServerError, "500 Internal Server Error: No webhook could be parsed")
		return
	}

	repository := webhook.Repository()
	l := logrus.WithFields(map[string]interface{}{
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
		err = o.onInstallHook(l, installHook)
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

	err = o.onGeneralHook(l, installRef, webhook)
	if err != nil {
		l.WithError(err).Error("failed to process webook")

		responseHTTPError(w, http.StatusInternalServerError, "500 Internal Server Error: %s", err.Error())
	}
	writeResult(l, w, "OK")
	return
}
