package hook

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/cloudbees/lighthouse-githubapp/pkg/flags"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/factory"
	"github.com/jenkins-x/go-scm/scm/transport"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	// HealthPath is the URL path for the HTTP endpoint that returns health status.
	HealthPath = "/health"
	// ReadyPath URL path for the HTTP endpoint that returns ready status.
	ReadyPath = "/ready"
)

type Options struct {
	Port           string
	Path           string
	Version        string
	PrivateKeyFile string
}

// health returns either HTTP 204 if the service is healthy, otherwise nothing ('cos it's dead).
func (o *Options) health(w http.ResponseWriter, r *http.Request) {
	logrus.Debug("Health check")
	w.WriteHeader(http.StatusNoContent)
}

// ready returns either HTTP 204 if the service is ready to serve requests, otherwise HTTP 503.
func (o *Options) ready(w http.ResponseWriter, r *http.Request) {
	logrus.Debug("Ready check")
	if o.isReady() {
		w.WriteHeader(http.StatusNoContent)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
}

func (o *Options) defaultHandler(w http.ResponseWriter, r *http.Request) {
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
func (o *Options) getIndex(w http.ResponseWriter, r *http.Request) {
	logrus.Debug("GET index")
	message := fmt.Sprintf(`Hello from Jenkins X Lighthouse version: %s

For more information see: https://github.com/jenkins-x/lighthouse
`, o.Version)

	_, err := w.Write([]byte(message))
	if err != nil {
		logrus.Debugf("failed to write the index: %v", err)
	}
}

func (o *Options) isReady() bool {
	// TODO a better readiness check
	return true
}

func (o *Options) handleWebHookRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		// liveness probe etc
		logrus.WithField("method", r.Method).Debug("invalid http method so returning index")
		o.getIndex(w, r)
		return
	}
	logrus.Infof("about to parse webhook")

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

	installRef := webhook.GetInstallationRef()
	if installRef == nil || installRef.ID == 0 {
		logrus.WithField("Hook", webhook).Error("no installation reference was passed for webhook")

		responseHTTPError(w, http.StatusInternalServerError, "500 Internal Server Error: No installation in webhook")
		return
	}

	repository := webhook.Repository()
	fields := map[string]interface{}{
		"Namespace": repository.Namespace,
		"Name":      repository.Name,
		"Branch":    repository.Branch,
		"Link":      repository.Link,
		"ID":        repository.ID,
		"Clone":     repository.Clone,
		"CloneSSH":  repository.CloneSSH,
	}

	l := logrus.WithFields(fields)

	installHook, ok := webhook.(*scm.InstallationHook)
	if ok {
		l.Info("invoking Installation handler")
		o.onInstallHook(l, installHook)
		return
	}

	pushHook, ok := webhook.(*scm.PushHook)
	if ok {
		fields["Ref"] = pushHook.Ref
		fields["BaseRef"] = pushHook.BaseRef
		fields["Commit.Sha"] = pushHook.Commit.Sha
		fields["Commit.Link"] = pushHook.Commit.Link
		fields["Commit.Author"] = pushHook.Commit.Author
		fields["Commit.Message"] = pushHook.Commit.Message
		fields["Commit.Committer.Name"] = pushHook.Commit.Committer.Name

		l.Info("invoking Push handler")
		o.onPushHook(pushHook)
		return
	}

	prHook, ok := webhook.(*scm.PullRequestHook)
	if ok {
		action := prHook.Action
		fields["Action"] = action.String()
		pr := prHook.PullRequest
		fields["PR.Number"] = pr.Number
		fields["PR.Ref"] = pr.Ref
		fields["PR.Sha"] = pr.Sha
		fields["PR.Title"] = pr.Title
		fields["PR.Body"] = pr.Body

		l.Info("invoking PR handler")
		o.onPullRequest(pushHook)
		return
	}

	branchHook, ok := webhook.(*scm.BranchHook)
	if ok {
		action := branchHook.Action
		ref := branchHook.Ref
		sender := branchHook.Sender
		fields["Action"] = action.String()
		fields["Ref.Sha"] = ref.Sha
		fields["Sender.Name"] = sender.Name

		l.Info("invoking branch handler")
		o.onBranch(pushHook)
		return
	}

	issueCommentHook, ok := webhook.(*scm.IssueCommentHook)
	if ok {
		action := issueCommentHook.Action
		issue := issueCommentHook.Issue
		comment := issueCommentHook.Comment
		sender := issueCommentHook.Sender
		fields["Action"] = action.String()
		fields["Issue.Number"] = issue.Number
		fields["Issue.Title"] = issue.Title
		fields["Issue.Body"] = issue.Body
		fields["Comment.Body"] = comment.Body
		fields["Sender.Body"] = sender.Name
		fields["Sender.Login"] = sender.Login
		fields["Kind"] = "IssueCommentHook"

		l.Info("invoking Issue Comment handler")
		o.onIssueComment(l, issueCommentHook)
		return
	}

	prCommentHook, ok := webhook.(*scm.PullRequestCommentHook)
	if ok {
		action := prCommentHook.Action
		fields["Action"] = action.String()
		pr := prCommentHook.PullRequest
		fields["PR.Number"] = pr.Number
		fields["PR.Ref"] = pr.Ref
		fields["PR.Sha"] = pr.Sha
		fields["PR.Title"] = pr.Title
		fields["PR.Body"] = pr.Body
		comment := prCommentHook.Comment
		fields["Comment.Body"] = comment.Body
		author := comment.Author
		fields["Author.Name"] = author.Name
		fields["Author.Login"] = author.Login
		fields["Author.Avatar"] = author.Avatar

		l.Info("invoking PR Comment handler")
		o.onPullRequestComment(l, issueCommentHook)
		return
	}

	l.Infof("unknown webhook %#v", webhook)
	_, err = w.Write([]byte("ignored unknown hook"))
	if err != nil {
		l.Debugf("failed to process the unknown hook")
	}
}

func (o *Options) createSCMClient(token string) (*scm.Client, string, string, error) {
	kind := flags.GitKind.Value()
	serverURL := flags.GitServer.Value()
	if token == "" {
		token = flags.GitToken.Value()
	}
	client, err := factory.NewClient(kind, serverURL, "")
	if err != nil {
		return client, serverURL, token, err
	}

	// add Apps installation token
	defaultScmTransport(client)
	tr := &transport.Custom{Base: &transport.Authorization{
		Base:        http.DefaultTransport,
		Scheme:      "token",
		Credentials: token,
	},
		Before: func(r *http.Request) {
			r.Header.Set("Accept", "application/vnd.github.machine-man-preview+json")
		},
	}
	client.Client.Transport = tr
	return client, serverURL, token, err
}

func (o *Options) onPushHook(hook *scm.PushHook) {

}

func (o *Options) onPullRequest(hook *scm.PushHook) {

}

func (o *Options) onBranch(hook *scm.PushHook) {

}

func (o *Options) onIssueComment(log *logrus.Entry, hook *scm.IssueCommentHook) error {
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

func (o *Options) onPullRequestComment(log *logrus.Entry, hook *scm.IssueCommentHook) error {
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

func (o *Options) createAppsScmClient() (*scm.Client, int, error) {
	scmClient, _, _, err := o.createSCMClient("")
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to create ScmClient")
	}
	defaultScmTransport(scmClient)
	appID := flags.GitHubAppID.Value()
	if err != nil {
		return nil, appID, fmt.Errorf("missing $LHA_APP_ID")
	}
	logrus.Infof("using GitHub App ID %d", appID)
	tr, err := ghinstallation.NewAppsTransportKeyFromFile(scmClient.Client.Transport, appID, o.PrivateKeyFile)
	if err != nil {
		return nil, appID, errors.Wrapf(err, "failed to create the Apps transport for AppID %v and file %s", appID, o.PrivateKeyFile)
	}
	scmClient.Client.Transport = tr
	return scmClient, appID, err
}

func defaultScmTransport(scmClient *scm.Client) {
	if scmClient.Client == nil {
		scmClient.Client = http.DefaultClient
	}
	if scmClient.Client.Transport == nil {
		scmClient.Client.Transport = http.DefaultTransport
	}
}

func (o *Options) onInstallHook(log *logrus.Entry, hook *scm.InstallationHook) {
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

func (o *Options) Handle(mux *http.ServeMux) {
	mux.Handle(HealthPath, http.HandlerFunc(o.health))
	mux.Handle(ReadyPath, http.HandlerFunc(o.ready))

	mux.Handle("/", http.HandlerFunc(o.defaultHandler))
	mux.Handle(o.Path, http.HandlerFunc(o.handleWebHookRequests))
}

func (o *Options) getInstallScmClient(log *logrus.Entry, ctx context.Context, ref *scm.InstallationRef) (*scm.Client, error) {
	// TODO use cache...
	if ref == nil || ref.ID == 0 {
		return nil, fmt.Errorf("missing installation.ID on webhok")
	}
	appsClient, _, err := o.createAppsScmClient()
	if err != nil {
		return nil, err
	}
	tokenResource, _, err := appsClient.Apps.CreateInstallationToken(ctx, ref.ID)
	if err != nil {
		log.WithError(err).Error("failed to create installation token")
		return nil, err
	}
	if tokenResource == nil {
		return nil, fmt.Errorf("no token returned from CreateInstallationToken()")
	}
	token := tokenResource.Token
	if token == "" {
		return nil, fmt.Errorf("empty token returned from CreateInstallationToken()")
	}
	client, _, _, err := o.createSCMClient(token)
	if err != nil {
		log.WithError(err).Error("failed to create real SCM client")
		return nil, err
	}
	return client, nil
}

func responseHTTPError(w http.ResponseWriter, statusCode int, response string) {
	logrus.WithFields(logrus.Fields{
		"response":    response,
		"status-code": statusCode,
	}).Info(response)
	http.Error(w, response, statusCode)
}
