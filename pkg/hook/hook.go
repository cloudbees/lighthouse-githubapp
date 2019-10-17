package hook

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/cloudbees/lighthouse-githubapp/pkg/flags"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/sirupsen/logrus"
)

const (
	// SetupPath URL path for the HTTP endpoint for the setup page
	SetupPath = "/setup"
	// HookPath URL path for the HTTP endpoint for handling webhooks
	HookPath = "/hook"
	// HealthPath is the URL path for the HTTP endpoint that returns health status.
	HealthPath = "/health"
	// ReadyPath URL path for the HTTP endpoint that returns ready status.
	ReadyPath = "/ready"
)

type HookOptions struct {
	Port           string
	Path           string
	Version        string
	PrivateKeyFile string
}

// NewHook create a new hook handler
func NewHook(privateKeyFile string) *HookOptions {
	return &HookOptions{
		Path:           HookPath,
		Port:           flags.HttpPort.Value(),
		Version:        "TODO",
		PrivateKeyFile: privateKeyFile,
	}
}

func (o *HookOptions) Handle(mux *http.ServeMux) {
	mux.Handle(HealthPath, http.HandlerFunc(o.health))
	mux.Handle(ReadyPath, http.HandlerFunc(o.ready))
	mux.Handle(SetupPath, http.HandlerFunc(o.setup))

	mux.Handle("/", http.HandlerFunc(o.defaultHandler))
	mux.Handle(o.Path, http.HandlerFunc(o.handleWebHookRequests))
}

// health returns either HTTP 204 if the service is healthy, otherwise nothing ('cos it's dead).
func (o *HookOptions) health(w http.ResponseWriter, r *http.Request) {
	logrus.Debug("Health check")
	w.WriteHeader(http.StatusNoContent)
}

// ready returns either HTTP 204 if the service is ready to serve requests, otherwise HTTP 503.
func (o *HookOptions) ready(w http.ResponseWriter, r *http.Request) {
	logrus.Debug("Ready check")
	if o.isReady() {
		w.WriteHeader(http.StatusNoContent)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
}

// setup handle the setup URL
func (o *HookOptions) setup(w http.ResponseWriter, r *http.Request) {
	logrus.Debug("setup")

	action := r.URL.Query().Get("setup_action")
	installationID := r.URL.Query().Get("installation_id")
	message := fmt.Sprintf(`Welcome to the Jenkins X Bot for installation: %s action: %s`, installationID, action)

	_, err := w.Write([]byte(message))
	if err != nil {
		logrus.Debugf("failed to write the setup: %v", err)
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
	logrus.Debug("GET index")
	message := fmt.Sprintf(`Hello from Jenkins X Lighthouse version: %s

For more information see: https://github.com/jenkins-x/lighthouse
`, o.Version)

	_, err := w.Write([]byte(message))
	if err != nil {
		logrus.Debugf("failed to write the index: %v", err)
	}
}

func (o *HookOptions) isReady() bool {
	// TODO a better readiness check
	return true
}

func (o *HookOptions) onPushHook(hook *scm.PushHook) {

}

func (o *HookOptions) onPullRequest(hook *scm.PushHook) {

}

func (o *HookOptions) onBranch(hook *scm.PushHook) {

}

func (o *HookOptions) onIssueComment(log *logrus.Entry, hook *scm.IssueCommentHook) error {
	log.Infof("Issue Comment: %s by %s", hook.Comment.Body, hook.Comment.Author.Login)
	author := hook.Comment.Author.Login
	if author != flags.BotName.Value() {
		ctx := context.Background()
		scmClient, err := o.getInstallScmClient(log, ctx, hook.GetInstallationRef())
		if err != nil {
			return err
		}

		prNumber := hook.Issue.Number
		repo := hook.Repository().FullName
		_, _, err = scmClient.PullRequests.CreateComment(ctx, repo, prNumber, &scm.CommentInput{
			Body: "hello from GitHub App",
		})
		if err != nil {
			log.WithError(err).Error("failed to comment on issue")
			return err
		} else {
			log.Infof("added comment")
		}
	}
	return nil
}

func (o *HookOptions) onPullRequestComment(log *logrus.Entry, hook *scm.IssueCommentHook) error {
	author := hook.Comment.Author.Login
	log.Infof("PR Comment: %s by %s", hook.Comment.Body, author)
	if author != flags.BotName.Value() {
		ctx := context.Background()
		scmClient, err := o.getInstallScmClient(log, ctx, hook.GetInstallationRef())
		if err != nil {
			return err
		}

		prNumber := hook.Issue.Number
		repo := hook.Repository().Namespace
		_, _, err = scmClient.PullRequests.CreateComment(ctx, repo, prNumber, &scm.CommentInput{
			Body: "hello from GitHub App",
		})
		if err != nil {
			log.WithError(err).Error("failed to comment on issue")
			return err
		} else {
			log.Infof("added comment")
		}
	}
	return nil
}

func (o *HookOptions) onInstallHook(log *logrus.Entry, hook *scm.InstallationHook) {
	install := hook.Installation
	id := install.ID
	account := install.Account
	fields := map[string]interface{}{
		"Action":           hook.Action.String(),
		"InstallationID":   id,
		"AccountID":        account.ID,
		"AccountLogin":     account.Login,
		"AccessTokensLink": install.AccessTokensLink,
	}
	log.WithFields(fields).Infof("installHook")

}

func responseHTTPError(w http.ResponseWriter, statusCode int, response string) {
	logrus.WithFields(logrus.Fields{
		"response":    response,
		"status-code": statusCode,
	}).Info(response)
	http.Error(w, response, statusCode)
}
