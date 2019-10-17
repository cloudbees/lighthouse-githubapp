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
