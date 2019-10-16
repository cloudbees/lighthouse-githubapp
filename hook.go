package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/cloudbees/lighthouse-githubapp/flags"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/factory"
	"github.com/jenkins-x/go-scm/scm/transport"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Options struct {
	Port           string
	Path           string
	Version        string
	PrivateKeyFile string
	installationID int64
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

	l := logrus.WithFields(logrus.Fields(fields))

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
		// lets add a dummy comment...
		ctx := context.Background()
		scmClient, _, err := o.createAppsScmClient()
		if err != nil {
			return err
		}
		installationID := o.installationID
		if installationID <= 0 {
			installationID = int64(flags.InstallationID.Value())
		}
		if installationID <= 0 {
			return fmt.Errorf("missing $LHA_INSTALLATION_ID")
		}

		tokenResource, _, err := scmClient.Apps.CreateInstallationToken(ctx, int64(installationID))
		if err != nil {
			log.WithError(err).Error("failed to create installation token")
			return err
		} else {
			if tokenResource != nil {
				token := tokenResource.Token
				log.Infof("got token %s", token)
				client, _, _, err := o.createSCMClient(token)
				if err != nil {
					log.WithError(err).Error("failed to create real SCM client")
					return err
				} else {
					prNumber := hook.Issue.Number
					repo := hook.Repository().FullName
					_, _, err = client.PullRequests.CreateComment(ctx, repo, prNumber, &scm.CommentInput{
						Body: "hello from GitHub App",
					})
					if err != nil {
						log.WithError(err).Error("failed to comment on issue")
					} else {
						log.Infof("added comment")
					}
				}
			}
		}
	}
	return nil
}

func (o *Options) onPullRequestComment(log *logrus.Entry, hook *scm.IssueCommentHook) error {
	log = log.WithField("InstallationID", o.installationID)
	author := hook.Comment.Author.Login
	log.Infof("PR Comment: %s by %s", hook.Comment.Body, author)

	if author != flags.BotName.Value() {
		// lets add a dummy comment...
		ctx := context.Background()

		scmClient, _, err := o.createAppsScmClient()
		if err != nil {
			return err
		}

		installationID := o.installationID
		if installationID <= 0 {
			installationID = int64(flags.InstallationID.Value())
		}
		if installationID <= 0 {
			return fmt.Errorf("missing $LHA_INSTALLATION_ID")
		}

		tokenResource, _, err := scmClient.Apps.CreateInstallationToken(ctx, int64(installationID))
		if err != nil {
			log.WithError(err).Error("failed to create installation token")
			return err
		} else {
			if tokenResource != nil {
				token := tokenResource.Token
				log.Infof("got token %s", token)
				client, _, _, err := o.createSCMClient(token)
				if err != nil {
					log.WithError(err).Error("failed to create real SCM client")
					return err
				} else {
					prNumber := hook.Issue.Number
					repo := hook.Repository().Namespace
					_, _, err = client.PullRequests.CreateComment(ctx, repo, prNumber, &scm.CommentInput{
						Body: "hello from GitHub App",
					})
					if err != nil {
						log.WithError(err).Error("failed to comment on issue")
					} else {
						log.Infof("added comment")
					}
				}
			}
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
		return nil, appID, errors.Wrapf(err, "failed to create the Apps transport for %v and file %s", o.installationID, o.PrivateKeyFile)
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
	if id != 0 {
		o.installationID = id
	}
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
