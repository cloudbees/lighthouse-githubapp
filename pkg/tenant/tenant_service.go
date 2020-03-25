package tenant

import (
	"context"
	"fmt"

	"github.com/cloudbees/jx-tenant-service/pkg/access"
	"github.com/cloudbees/jx-tenant-service/pkg/client"
	"github.com/cloudbees/jx-tenant-service/pkg/clientutils"
	"github.com/cloudbees/jx-tenant-service/pkg/domain"
	"github.com/cloudbees/jx-tenant-service/pkg/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type tenantService struct {
	client *client.Client
}

func NewTenantService(host string) *tenantService {
	c := clientutils.NewClientForHost("")
	return &tenantService{
		client: c,
	}
}

// AppInstall registers an app installation on a number of repos
func (t *tenantService) AppInstall(ctx context.Context, log *logrus.Entry, installationID int64, ownerURL string) error {
	path := installationPath(installationID)
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
func (t *tenantService) AppUnnstall(ctx context.Context, log *logrus.Entry, installationID int64) error {
	path := installationPath(installationID)
	_, err := t.client.DeleteGitHubAppInstallGithubApp(ctx, path)
	if err != nil {
		log.WithError(err).Error("failed to uninstall app")
		return err
	}
	log.Infof("removed Installation")
	return nil
}

func (t *tenantService) FindWorkspaces(ctx context.Context, log *logrus.Entry, installationID int64, gitURL string) ([]*access.WorkspaceAccess, error) {
	path := client.GetRepositoryWorkspacesWorkspacesPath()
	installation := model.Int64ToA(installationID)
	resp, err := t.client.GetRepositoryWorkspacesWorkspaces(ctx, path, &gitURL, &installation)
	if err != nil {
		log.WithError(err).Error("failed to uninstall app")
		return nil, err
	}
	results, err := t.client.DecodeWorkspaceAccessCollection(resp)
	if err != nil {
		log.WithError(err).Error("failed to unmarshall the response")
		return nil, err
	}
	return clientutils.ToWorkspaceAccesses(results), nil
}

// GetGithubAppToken returns the github app token for the installation
func (t *tenantService) GetGithubAppToken(ctx context.Context, log *logrus.Entry, installationID int64) (*domain.InstallationToken, error) {
	installation := model.Int64ToA(installationID)
	path := client.GetGithubAppTokenWorkspacesPath(installation)
	resp, err := t.client.GetGithubAppTokenWorkspaces(ctx, path)
	if err != nil {
		err = errors.Wrapf(err, "failed to get GitHub App token")
		log.WithError(err).Error(err.Error())
		return nil, err
	}
	gitToken, err := t.client.DecodeGitToken(resp)
	if err != nil {
		err = errors.Wrapf(err, "failed to unmarshall the response")
		log.WithError(err).Error(err.Error())
		return nil, err
	}
	if gitToken == nil {
		err = fmt.Errorf("no GitHub App token returned")
		log.WithError(err).Error(err.Error())
		return nil, err
	}
	if gitToken.Token == nil || *gitToken.Token == "" {
		err = fmt.Errorf("empty GitHub App token returned")
		log.WithError(err).Error(err.Error())
		return nil, err
	}
	return clientutils.ToInstallationToken(gitToken), nil
}

func installationPath(installationID int64) string {
	return client.CreateGitHubAppInstallGithubAppPath(model.Int64ToA(installationID))
}
