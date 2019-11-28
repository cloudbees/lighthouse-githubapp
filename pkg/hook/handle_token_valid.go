package hook

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/sirupsen/logrus"
)

func (o *HookOptions) handleTokenValid(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		responseHTTPError(w, http.StatusInternalServerError, fmt.Sprintf("500 Internal Server Error: Invalid method: %s", r.Method))
		return
	}

	vars := mux.Vars(r)
	installation := vars["installation"]
	if installation == "" {
		responseHTTPError(w, http.StatusInternalServerError, fmt.Sprintf("500 Internal Server Error: Missing installation"))
		return
	}

	id, err := ParseInt64(installation)
	if installation == "" {
		responseHTTPError(w, http.StatusInternalServerError, fmt.Sprintf("500 Internal Server Error: Failed to parse numeric installation: %s due to: %s", installation, err.Error()))
		return
	}

	l := logrus.WithField("Installation", id)

	ctx := context.Background()
	installRef := &scm.InstallationRef{
		ID: id,
	}
	scmClient, _, err := o.getInstallScmClient(l, ctx, installRef)
	if err != nil {
		responseHTTPError(w, http.StatusInternalServerError, fmt.Sprintf("500 Internal Server Error: Failed to create Scm client from installation: %s due to: %s", installation, err.Error()))
		return
	}

	// now lets query the something
	getPRLabels(l, scmClient, ctx, w, installation)
}

func getPRLabels(l *logrus.Entry, scmClient *scm.Client, ctx context.Context, w http.ResponseWriter, installation string) {
	labels, _, err := scmClient.PullRequests.ListLabels(ctx, verifyRepository, verifyPullRequest, scm.ListOptions{Size: 100})
	if err != nil {
		responseHTTPError(w, http.StatusInternalServerError, fmt.Sprintf("500 Internal Server Error: Failed to list PR labesl for installation: %s due to: %s", installation, err.Error()))
		return
	}
	buffer := strings.Builder{}
	buffer.WriteString(fmt.Sprintf("Found labels on PR %s %d:\n", verifyRepository, verifyPullRequest))
	for _, label := range labels {
		buffer.WriteString(label.Name)
		buffer.WriteString("\n")
	}
	writeResult(l, w, buffer.String())
}

func listRepositories(l *logrus.Entry, scmClient *scm.Client, ctx context.Context, w http.ResponseWriter, installation string) {
	repos, _, err := scmClient.Repositories.List(ctx, scm.ListOptions{Size: 100})
	if err != nil {
		responseHTTPError(w, http.StatusInternalServerError, fmt.Sprintf("500 Internal Server Error: Failed to list repositories for installation: %s due to: %s", installation, err.Error()))
		return
	}
	buffer := strings.Builder{}
	buffer.WriteString("Found repositories:\n")
	for _, org := range repos {
		buffer.WriteString(org.Name)
		buffer.WriteString("\n")
	}
	writeResult(l, w, buffer.String())
}

func listOrganisations(l *logrus.Entry, scmClient *scm.Client, ctx context.Context, w http.ResponseWriter, installation string) {
	orgs, _, err := scmClient.Organizations.List(ctx, scm.ListOptions{Size: 100})
	if err != nil {
		responseHTTPError(w, http.StatusInternalServerError, fmt.Sprintf("500 Internal Server Error: Failed to list organisations for installation: %s due to: %s", installation, err.Error()))
		return
	}
	buffer := strings.Builder{}
	buffer.WriteString("Found organisations:\n")
	for _, org := range orgs {
		buffer.WriteString(org.Name)
		buffer.WriteString("\n")
	}
	writeResult(l, w, buffer.String())
}
