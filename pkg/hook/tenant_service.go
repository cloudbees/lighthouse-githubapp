package hook

import (
	"context"

	"github.com/cloudbees/jx-tenant-service/pkg/client"
	"github.com/cloudbees/jx-tenant-service/pkg/model"
	"github.com/sirupsen/logrus"
)

type TenantService struct {
	host   string
	client *client.Client
}

func NewTenantService(host string) *TenantService {
	c := client.NewClientForHost("")
	return &TenantService{
		client: c,
	}
}

// AppInstall registers an app installation on a number of repos
func (t *TenantService) AppInstall(log *logrus.Entry, installationID int64, ownerURL string) error {
	path := installationPath(installationID)
	ctx := context.Background()
	payload := &client.InstallAppRequest{
		OwnerURL: &ownerURL,
	}
	_, err := t.client.CreateGitHubAppInstallGithubApp(ctx, path, payload)
	if err != nil {
		log.WithError(err).Error("failed to install app")
		return err
	}
	log.Infof("added Installation")
	return nil
}

// AppUnnstall removes an App installation
func (t *TenantService) AppUnnstall(log *logrus.Entry, installationID int64) error {
	path := installationPath(installationID)
	ctx := context.Background()

	_, err := t.client.DeleteGitHubAppInstallGithubApp(ctx, path)
	if err != nil {
		log.WithError(err).Error("failed to uninstall app")
		return err
	}
	log.Infof("removed Installation")
	return nil
}

func (t *TenantService) FindWorkspaces(log *logrus.Entry, installationID int64, gitURL string) ([]WorkspaceAccess, error) {
	path := client.GetRepositoryWorkspacesWorkspacesPath()
	ctx := context.Background()
	installation := model.Int64ToA(installationID)
	_, err := t.client.GetRepositoryWorkspacesWorkspaces(ctx, path, &gitURL, &installation)
	if err != nil {
		log.WithError(err).Error("failed to uninstall app")
		return nil, err
	}
	// TODO how to get the data back?
	return nil, nil
}

func installationPath(installationID int64) string {
	return client.CreateGitHubAppInstallGithubAppPath(model.Int64ToA(installationID))
}
