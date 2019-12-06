package hook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/sirupsen/logrus"
)

type GithubApp struct {
	ctx context.Context
}

type GithubAppResponse struct {
	Installed    bool
	AccessToRepo bool
	URL          string
	AppName      string
}

func NewGithubApp() (*GithubApp, error) {
	ctx := context.Background()

	logrus.Info("Initializing Github App")
	return &GithubApp{
		ctx,
	}, nil
}

func (o *GithubApp) handleInstalledRequests(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	owner := vars["owner"]
	repository := vars["repository"]
	l := logrus.WithField("Repository", repository).WithField("Owner", owner)

	l.Debugf("request received for owner %s and repository %s", owner, repository)

	scmClient, _, err := createAppsScmClient()
	if err != nil {
		logrus.Errorf("error creating Apps SCM client %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	githubAppResponse := &GithubAppResponse{}

	installation, response, err := o.findRepositoryInstallation(scmClient, owner, repository)

	if o.hasErrored(response, err) {
		l.Errorf("error from repository installation %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if response.Status == 404 {
		l.Debugf("didn't find the installation via the repository trying organisation")
		installation, response, err = scmClient.Apps.GetOrganisationInstallation(o.ctx, owner)
		if o.hasErrored(response, err) {
			l.Errorf("error from repository installation %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if response.Status == 404 {
			logrus.Debugf("didn't find the installation via the organisation trying the user account")
			installation, response, err = scmClient.Apps.GetUserInstallation(o.ctx, owner)
			if o.hasErrored(response, err) {
				l.Errorf("error from repository installation %v", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if response.Status == 404 {
				l.Debugf("didn't find the installation via the user account - github app not installed")
				githubAppResponse.Installed = false
				githubAppResponse.AccessToRepo = false
				githubAppResponse.URL = getGitHubAppInstalltionURL()
				githubAppResponse.AppName = getGitHubAppName()
			} else {
				githubAppResponse.Installed = true
				githubAppResponse.AccessToRepo = false
				githubAppResponse.URL = installation.Link
				githubAppResponse.AppName = getGitHubAppName()
			}
		} else {
			githubAppResponse.Installed = true
			githubAppResponse.AccessToRepo = false
			githubAppResponse.URL = installation.Link
			githubAppResponse.AppName = getGitHubAppName()
		}
	} else {
		githubAppResponse.Installed = true
		githubAppResponse.AccessToRepo = true
		githubAppResponse.URL = installation.Link
		githubAppResponse.AppName = getGitHubAppName()
	}

	res, err := json.Marshal(githubAppResponse)
	if err != nil {
		l.Errorf("failed to marshall struct to json: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = w.Write([]byte(res))
	if err != nil {
		l.Errorf("failed to write the message: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (o *GithubApp) hasErrored(response *scm.Response, err error) bool {
	if err != nil {
		logrus.Debugf("Determine if error is an issue %v", err)
		if response == nil {
			logrus.Errorf("error response %v", err)
			return true
		} else if response.Status == 200 || response.Status == 404 {
			logrus.Debugf("error is %v and response status is %q", err, response.Status)
			return false
		} else {
			logrus.Errorf("error response received status code %d with response %q", response.Status, response.Body)
			return true
		}
	}
	logrus.Debugf("Response received from github api %v", response)
	return false
}

func (o *GithubApp) findRepositoryInstallation(scmClient *scm.Client, owner string, repository string) (*scm.Installation, *scm.Response, error) {
	fullName := owner + "/" + repository
	return scmClient.Apps.GetRepositoryInstallation(o.ctx, fullName)
}

func getGitHubAppInstalltionURL() string {
	return fmt.Sprintf("https://github.com/apps/%s/installations/new", getBotName())
}

func getBotName() string {
	botEnvVar := os.Getenv("BOT_NAME")
	if botEnvVar != "" {
		botEnvVar = strings.ReplaceAll(botEnvVar, "[bot]", "")
	}

	if botEnvVar == "" {
		botEnvVar = "jenkins-x"
	}

	return botEnvVar
}

func getGitHubAppName() string {
	botName := strings.ReplaceAll(getBotName(), "-", " ")

	return strings.Title(strings.ToLower(botName))
}
