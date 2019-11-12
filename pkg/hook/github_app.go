package hook

import (
	"context"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/sirupsen/logrus"
	"net/http"
	"strconv"
)

type GithubApp struct {
	ctx context.Context
	scmClient *scm.Client

}

type GithubAppResponse struct {
	Installed    bool
	AccessToRepo bool
	URL          string
}

func NewGithubApp() (*GithubApp, error)  {
	ctx := context.Background()
	scmClient, _, err := createAppsScmClient()
	if err != nil {
		logrus.Errorf("error creating Apps SCM client %v", err)
		return nil, err
	}
	logrus.Info("Initializing Github App")
	return &GithubApp{
		ctx,
		scmClient,
	}, nil
}

func (o *GithubApp) handleInstalledRequests(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	owner := vars["owner"]
	repository := vars["repository"]

	githubAppResponse := &GithubAppResponse{}

	logrus.Debugf("request received for owner %s and repository %s", owner, repository)
	installation, response, err := o.findRepositoryInstallation(owner, repository)
	if o.hasErrored(response, err) {
		logrus.Errorf("error from repository installation %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if response.Status == 404 {
		logrus.Debugf("didn't find the installation via the repository trying organisation")
		installation, response, err = o.scmClient.Apps.GetOrganisationInstallation(o.ctx, owner)
		if o.hasErrored(response, err) {
			logrus.Errorf("error from repository installation %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if response.Status == 404 {
			logrus.Debugf("didn't find the installation via the organisation trying the user account")
			installation, response, err = o.scmClient.Apps.GetUserInstallation(o.ctx, owner)
			if o.hasErrored(response, err) {
				logrus.Errorf("error from repository installation %v", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if response.Status == 404 {
				logrus.Debugf("didn't find the installation via the user account - github app not installed")
				githubAppResponse.Installed = false
				githubAppResponse.AccessToRepo = false
				githubAppResponse.URL = "https://github.com/apps/jenkins-x/installations/new"
			} else {
				githubAppResponse.Installed = true
				githubAppResponse.AccessToRepo = false
				orgId := installation.ID
				githubAppResponse.URL = "https://github.com/settings/installations/" + strconv.FormatInt(orgId, 10)
			}
		} else {
			githubAppResponse.Installed = true
			githubAppResponse.AccessToRepo = false
			orgId := installation.ID
			githubAppResponse.URL = "https://github.com/settings/installations/" + strconv.FormatInt(orgId, 10)
		}
	} else {
		githubAppResponse.Installed = true
		githubAppResponse.AccessToRepo = true
		githubAppResponse.URL = installation.Link
	}

	res, err := json.Marshal(githubAppResponse)
	if err != nil {
		logrus.Errorf("failed to marshall struct to json: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = w.Write([]byte(res))
	if err != nil {
		logrus.Errorf("failed to write the message: %v", err)
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
			logrus.Debugf("error is %v and response statis is %q", err, response.Status)
			return false
		} else {
			logrus.Errorf("error response received status code %d with response %q", response.Status, response.Body)
			return true
		}
	}
	logrus.Debugf("Response received from github api %v", response)
	return false
}

func (o *GithubApp) findRepositoryInstallation(owner string, repository string) (*scm.Installation, *scm.Response, error) {
	fullName := owner + "/" + repository
	return o.scmClient.Apps.GetRepositoryInstallation(o.ctx, fullName)
}
