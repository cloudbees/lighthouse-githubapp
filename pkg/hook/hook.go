package hook

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/cloudbees/jx-tenant-service/pkg/access"
	"github.com/cloudbees/lighthouse-githubapp/pkg/hmac"
	"github.com/cloudbees/lighthouse-githubapp/pkg/tenant"

	"github.com/cloudbees/lighthouse-githubapp/pkg/util"

	"github.com/cloudbees/lighthouse-githubapp/pkg/flags"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	muxtrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/gorilla/mux"
)

const (
	repoNotConfiguredMessage = "repository not configured"
)

var (
	defaultMaxRetryDuration = 30 * time.Second
)

type HookOptions struct {
	Port             string
	Path             string
	Version          string
	tokenCache       *cache.Cache
	tenantService    tenant.TenantService
	githubApp        ghaClient
	secretFn         func(webhook scm.Webhook) (string, error)
	client           *http.Client
	maxRetryDuration *time.Duration
}

// NewHook create a new hook handler
func NewHook() (*HookOptions, error) {
	tokenCache := cache.New(tokenCacheExpiration, tokenCacheExpiration)
	tenantService := tenant.NewTenantService("")
	githubApp, err := NewGithubApp()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create hook")
	}

	secretFn := func(webhook scm.Webhook) (string, error) {
		return flags.HmacToken.Value(), nil
	}

	return &HookOptions{
		Path:             HookPath,
		Port:             flags.HttpPort.Value(),
		Version:          "TODO",
		tokenCache:       tokenCache,
		tenantService:    tenantService,
		githubApp:        githubApp,
		secretFn:         secretFn,
		maxRetryDuration: &defaultMaxRetryDuration,
	}, nil
}

func (o *HookOptions) Handle(mux *muxtrace.Router) {
	mux.Handle(GitHubAppPathWithoutRepository, http.HandlerFunc(o.githubApp.handleInstalledRequests))
	mux.Handle(GithubAppPath, http.HandlerFunc(o.githubApp.handleInstalledRequests))
	mux.Handle(TestTokenPath, http.HandlerFunc(o.handleTokenValid))
	mux.Handle(HealthPath, http.HandlerFunc(o.health))
	mux.Handle(ReadyPath, http.HandlerFunc(o.ready))
	mux.Handle(SetupPath, http.HandlerFunc(o.setup))

	mux.Handle("/", http.HandlerFunc(o.defaultHandler))
	mux.Handle(o.Path, http.HandlerFunc(o.handleWebHookRequests))
}

// health returns either HTTP 204 if the service is healthy, otherwise nothing ('cos it's dead).
func (o *HookOptions) health(w http.ResponseWriter, r *http.Request) {
	util.TraceLogger(r.Context()).Info("Health check")
	w.WriteHeader(http.StatusNoContent)
}

// ready returns either HTTP 204 if the service is ready to serve requests, otherwise HTTP 503.
func (o *HookOptions) ready(w http.ResponseWriter, r *http.Request) {
	util.TraceLogger(r.Context()).Debug("Ready check")
	if o.isReady() {
		w.WriteHeader(http.StatusNoContent)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
}

// setup handle the setup URL
func (o *HookOptions) setup(w http.ResponseWriter, r *http.Request) {
	util.TraceLogger(r.Context()).Debug("setup")

	action := r.URL.Query().Get("setup_action")
	installationID := r.URL.Query().Get("installation_id")
	message := fmt.Sprintf(`Welcome to the Jenkins X Bot for installation: %s action: %s`, installationID, action)

	_, err := w.Write([]byte(message))
	if err != nil {
		util.TraceLogger(r.Context()).Debugf("failed to write the setup: %v", err)
	}
}

func (o *HookOptions) defaultHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == o.Path || strings.HasPrefix(path, o.Path+"/") {
		o.handleWebHookRequests(w, r)
		return
	}
	path = strings.TrimPrefix(path, "/")
	if path == "" || path == "index.html" {
		o.getIndex(w, r)
		return
	}
	http.Error(w, fmt.Sprintf("unknown path %s", path), 404)
}

// getIndex returns a simple home page
func (o *HookOptions) getIndex(w http.ResponseWriter, r *http.Request) {
	l := util.TraceLogger(r.Context())
	l.Debug("GET index")
	message := fmt.Sprintf(`Hello from Jenkins X Lighthouse version: %s
`, o.Version)

	_, err := w.Write([]byte(message))
	if err != nil {
		l.Debugf("failed to write the index: %v", err)
	}
}

func (o *HookOptions) isReady() bool {
	// TODO a better readiness check
	return true
}

func (o *HookOptions) onInstallHook(ctx context.Context, log *logrus.Entry, hook *scm.InstallationHook) error {
	install := hook.Installation
	id := install.ID
	fields := map[string]interface{}{
		"Action":         hook.Action.String(),
		"InstallationID": id,
		"Function":       "onInstallHook",
	}
	log = log.WithFields(fields)

	// ets register / unregister repositories to the InstallationID
	if hook.Action == scm.ActionCreate {
		ownerURL := hook.Installation.Account.Link
		log = log.WithField("Owner", ownerURL)
		if ownerURL == "" {
			err := fmt.Errorf("missing ownerURL on install webhook")
			log.Error(err.Error())
			return err
		}

		return o.tenantService.AppInstall(ctx, log, id, ownerURL)
	} else if hook.Action == scm.ActionDelete {
		return o.tenantService.AppUnnstall(ctx, log, id)
	} else {
		log.Warnf("ignore unknown action")
		return nil
	}
}

func (o *HookOptions) onGeneralHook(ctx context.Context, log *logrus.Entry, install *scm.InstallationRef, webhook scm.Webhook, githubEventType string, githubDeliveryEvent string, bodyBytes []byte) error {
	// Set a default max retry duration of 30 seconds if it's not set.
	if o.maxRetryDuration == nil {
		o.maxRetryDuration = &defaultMaxRetryDuration
	}

	id := install.ID
	repo := webhook.Repository()

	// TODO this should be fixed in go-scm
	if repo.FullName == "" {
		log.Debugf("repo.FullName is empty, calculating full name as '%s/%s'", repo.Namespace, repo.Name)
		repo.FullName = repo.Namespace + "/" + repo.Name
	}
	fields := map[string]interface{}{
		"InstallationID": id,
		"FullName":       repo.FullName,
		"Link":           repo.Link,
	}
	log = log.WithFields(fields)
	u := repo.Link
	if u == "" {
		log.Warnf("ignoring webhook '%s' as no repository URL for '%s'", webhook.Kind(), repo.FullName)
		return nil
	}

	log.Debugf("onGeneralHook - %+v", webhook)
	var workspaces []*access.WorkspaceAccess

	getWsFunc := func() error {
		ws, err := o.tenantService.FindWorkspaces(ctx, log, id, u)
		if err != nil {
			log.WithError(err).Errorf("Unable to find workspaces for %s", repo.FullName)
			return err
		}
		log.Infof("%d workspaces interested in repository %s", len(ws), repo.FullName)

		if len(ws) == 0 {
			return errors.Errorf("no workspaces interested in repository '%s', backing off...", repo.FullName)
		}
		workspaces = append(workspaces, ws...)
		return nil
	}

	err := o.retryGetWorkspaces(getWsFunc, func(e error, d time.Duration) {
		log.Infof("get workspaces failed with '%s', backing off for %s", e, d)
	})
	if err != nil {
		log.WithError(err).Errorf("failed to find any workspaces after %s seconds for '%s'", o.maxRetryDuration, repo.FullName)
		return err
	}

	for _, ws := range workspaces {
		log := log.WithFields(ws.LogFields())
		log.Infof("notifying workspace %s for %s", ws.Project, repo.FullName)

		// TODO insecure webhooks should be configured on workspace creation and passed to this function
		useInsecureRelay := ShouldUseInsecureRelay(ws)

		log.Infof("invoking webhook relay here! url=%s, insecure=%t", ws.LighthouseURL, useInsecureRelay)

		decodedHmac, err := base64.StdEncoding.DecodeString(ws.HMAC)
		if err != nil {
			log.WithError(err).Errorf("unable to decode hmac")
			continue
		}

		err = o.retryWebhookDelivery(ws.LighthouseURL, githubEventType, githubDeliveryEvent, decodedHmac, useInsecureRelay, bodyBytes, log)
		if err != nil {
			log.WithError(err).Errorf("failed to deliver webhook after %s", o.maxRetryDuration)
			continue
		}
		log.Infof("webhook delivery ok for %s", repo.FullName)
	}

	return nil
}

// retryWebhookDelivery attempts to deliver the relayed webhook, but will retry a few times if the response is a 500 with
// "repository not configured" in the body, in case the remote Lighthouse doesn't yet have this repository in its configuration.
func (o *HookOptions) retryWebhookDelivery(lighthouseURL string, githubEventType string, githubDeliveryEvent string, decodedHmac []byte, useInsecureRelay bool, bodyBytes []byte, log *logrus.Entry) error {
	f := func() error {
		log.Debugf("relaying %s", string(bodyBytes))
		g := hmac.NewGenerator(decodedHmac)
		signature := g.HubSignature(bodyBytes)

		var httpClient *http.Client

		if o.client != nil {
			httpClient = o.client
		} else {
			if useInsecureRelay {
				tr := &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				}

				httpClient = &http.Client{Transport: tr}
			} else {
				httpClient = &http.Client{}
			}
		}

		req, err := http.NewRequest("POST", lighthouseURL, bytes.NewReader(bodyBytes))
		req.Header.Add("X-GitHub-Event", githubEventType)
		req.Header.Add("X-GitHub-Delivery", githubDeliveryEvent)
		req.Header.Add("X-Hub-Signature", signature)

		resp, err := httpClient.Do(req)
		log.Infof("got response code %d from url '%s',err=%s", resp.StatusCode, lighthouseURL, err)
		if err != nil {
			return err
		}

		// If we got a 500, check if it's got the "repository not configured" string in the body. If so, we retry.
		if resp.StatusCode == 500 {
			respBody, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return backoff.Permanent(errors.Wrap(err, "parsing response body"))
			}
			err = resp.Body.Close()
			if err != nil {
				return backoff.Permanent(errors.Wrap(err, "closing response body"))
			}
			log.Infof("got error response body '%s'", string(respBody))
			if strings.Contains(string(respBody), repoNotConfiguredMessage) {
				return errors.New("repository not configured in Lighthouse")
			}
		}

		// If we got anything other than a 2xx, retry as well.
		// We're leaving this distinct from the "not configured" behavior in case we want to resurrect that later. (apb)
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return errors.Errorf("%s not available, error was %s", req.URL.String(), resp.Status)
		}

		// And finally, if we haven't gotten any errors, just return nil because we're good.
		return nil
	}

	bo := backoff.NewExponentialBackOff()
	// Try again after 2/4/8/... seconds if necessary, for up to 30 seconds.
	bo.InitialInterval = 2 * time.Second
	bo.MaxElapsedTime = *o.maxRetryDuration
	bo.Reset()

	return backoff.RetryNotify(f, bo, func(e error, t time.Duration) {
		log.Infof("webhook relaying failed: %s, backing off for %s", e, t)
	})
}

func (o *HookOptions) retryGetWorkspaces(f func() error, n func(error, time.Duration)) error {
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = *o.maxRetryDuration
	bo.Reset()
	return backoff.RetryNotify(f, bo, n)
}

func ShouldUseInsecureRelay(ws *access.WorkspaceAccess) bool {
	return strings.Contains(ws.LighthouseURL, "-pr-") ||
		strings.Contains(ws.LighthouseURL, ".play-jxaas.live") ||
		strings.Contains(ws.LighthouseURL, ".staging-jxaas.live")
}
