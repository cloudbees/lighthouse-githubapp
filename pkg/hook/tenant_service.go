package hook

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"github.com/cloudbees/lighthouse-githubapp/pkg/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	AppInstallationPath = "/api/v1/github/app/installation"
	AppWorkspacesPath   = "/api/v1/github/app/workspaces"
)

type TenantService struct {
	host string
}

// WorkspaceAccess returns details of the rocket for connecting to it
type WorkspaceAccess struct {
	Project   string `json:"project,omitempty"`
	Cluster   string `json:"cluster,omitempty"`
	Region    string `json:"region,omitempty"`
	Zone      string `json:"zone,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

// LogFields returns fields for logging
func (w *WorkspaceAccess) LogFields() map[string]interface{} {
	return map[string]interface{}{
		"Project": w.Project,
		"Cluster": w.Cluster,
		"Region":  w.Region,
	}
}

type RepositoryInfo struct {
	URL string `json:"url,omitempty"`
}

// GetTenantServiceURL returns the tenant service URL with the given
func (t *TenantService) GetTenantServiceURL(path string) string {
	if t.host == "" {
		t.host = os.Getenv("JX_TENANT_SERVICE")
		if t.host == "" {
			t.host = "http://jx-internal-tenant-service"
		}
	}
	if path == "" {
		path = AppInstallationPath
	}
	return util.UrlJoin(t.host, path)
}

// AppInstall registers an app installation on a number of repos
func (t *TenantService) AppInstall(log *logrus.Entry, installationID int64, repos []RepositoryInfo) error {

	u := t.getAppInstallURL(installationID)
	log = log.WithFields(map[string]interface{}{
		"Function":     "AppInstall",
		"URL":          u,
		"Installation": installationID,
	})

	data, err := json.Marshal(repos)
	if err != nil {
		err = errors.Wrapf(err, "failed to marshal payload")
		log.WithError(err).Error(err.Error())
		return err
	}
	req, err := http.NewRequest(http.MethodPost, u, bytes.NewReader(data))
	if err != nil {
		err = errors.Wrapf(err, "failed to create http request")
		log.WithError(err).Error(err.Error())
		return err
	}
	err = doHTTPRequest(log, req)
	if err != nil {
		return err
	}
	log.Infof("added Installation")
	return nil
}

// AppUnnstall removes an App installation
func (t *TenantService) AppUnnstall(log *logrus.Entry, installationID int64) error {
	u := t.getAppInstallURL(installationID)
	log = log.WithFields(map[string]interface{}{
		"Function":     "AppUnnstall",
		"URL":          u,
		"Installation": installationID,
	})
	req, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		err = errors.Wrapf(err, "failed to create http request")
		log.WithError(err).Error(err.Error())
		return err
	}
	err = doHTTPRequest(log, req)
	if err != nil {
		return err
	}
	log.Infof("removed Installation")
	return nil
}

func doHTTPRequest(log *logrus.Entry, req *http.Request) error {
	req.Header.Set("Content-Type", "application/json")
	httpClient := util.GetClient()
	resp, err := httpClient.Do(req)
	if resp == nil && err == nil {
		err = errors.New("no response returned from tenant service for request")
	}
	if err != nil {
		log.WithError(err).Error(err.Error())
		return err
	}
	statusCode := resp.StatusCode
	status := resp.Status
	log = log.WithField("StatusCode", statusCode).WithField("Status", status)
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = errors.Wrap(err, "loading response body")
		log.WithError(err).Error(err.Error())
		return err
	}
	resp.Body.Close()
	if statusCode < 200 && statusCode >= 300 {
		log.Errorf("failed: %s", string(data))
		return errors.Errorf("error status: %d %s", statusCode, status)
	}
	log.Infof("success: %s", string(data))
	return nil
}

func (t *TenantService) getAppInstallURL(installationID int64) string {
	path := util.UrlJoin(AppInstallationPath, strconv.FormatInt(installationID, 10))
	return t.GetTenantServiceURL(path)
}

func (t *TenantService) FindWorkspaces(i int64, s string) ([]WorkspaceAccess, error) {
	// TODO
	return nil, nil
}
