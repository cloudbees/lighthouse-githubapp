package hook

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/cloudbees/lighthouse-githubapp/pkg/flags"
	"github.com/cloudbees/lighthouse-githubapp/pkg/util"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
)

type HookOptions struct {
	Port           string
	Path           string
	Version        string
	PrivateKeyFile string
	tokenCache     *cache.Cache
	tenantService  *TenantService
}

// NewHook create a new hook handler
func NewHook(privateKeyFile string) *HookOptions {
	tokenCache := cache.New(tokenCacheExpiration, tokenCacheExpiration)

	return &HookOptions{
		Path:           HookPath,
		Port:           flags.HttpPort.Value(),
		Version:        "TODO",
		PrivateKeyFile: privateKeyFile,
		tokenCache:     tokenCache,
		tenantService:  &TenantService{},
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

func (o *HookOptions) onInstallHook(log *logrus.Entry, hook *scm.InstallationHook) error {
	install := hook.Installation
	id := install.ID
	fields := map[string]interface{}{
		"Action":         hook.Action.String(),
		"InstallationID": id,
	}
	log = log.WithFields(fields)
	log.Infof("installHook")

	// ets register / unregister repositories to the InstallationID
	if hook.Action == scm.ActionCreate {
		repos := []RepositoryInfo{}
		for _, repo := range hook.Repos {
			link := strings.TrimSuffix(repo.Link, "/")
			link = strings.TrimSuffix(link, ".git")
			if link == "" {
				full := repo.FullName
				if full != "" {
					link = util.UrlJoin("https://github.com", full)
				}
			}
			if link != "" {
				repos = append(repos, RepositoryInfo{URL: link})
			}
		}
		if len(repos) == 0 {

		}
		return o.tenantService.AppInstall(log, id, repos)
	} else if hook.Action == scm.ActionDelete {
		return o.tenantService.AppUnnstall(log, id)
	} else {
		log.Warnf("ignore unknown action")
		return nil
	}
}

func (o *HookOptions) onGeneralHook(log *logrus.Entry, install *scm.InstallationRef, webhook scm.Webhook) error {
	id := install.ID
	fields := map[string]interface{}{
		"InstallationID": id,
	}
	log = log.WithFields(fields)
	log.Infof("onGeneralHook")

	repo := webhook.Repository()
	u := repo.Link
	if u == "" {
		log.WithField("Fullname", repo.FullName).Warnf("ignoring webhook as no repository URL for")
		return nil
	}
	workspaces, err := o.tenantService.FindWorkspaces(id, u)
	if err != nil {
		return err
	}

	for _, ws := range workspaces {
		log.WithFields(ws.LogFields()).Infof("got workspace")

		// TODO now use it!
	}
	return nil
}
