package tenant

import (
	"context"

	"github.com/cloudbees/jx-tenant-service/pkg/access"
	"github.com/cloudbees/jx-tenant-service/pkg/domain"
	"github.com/sirupsen/logrus"
)

type fakeTenantService struct {
	workspace *access.WorkspaceAccess
}

func NewFakeTenantService(w *access.WorkspaceAccess) *fakeTenantService {
	return &fakeTenantService{w}
}

// AppInstall registers an app installation on a number of repos
func (t *fakeTenantService) AppInstall(ctx context.Context, log *logrus.Entry, installationID int64, ownerURL string) error {
	return nil
}

// AppUnnstall removes an App installation
func (t *fakeTenantService) AppUnnstall(ctx context.Context, log *logrus.Entry, installationID int64) error {
	return nil
}

func (t *fakeTenantService) FindWorkspaces(ctx context.Context, log *logrus.Entry, installationID int64, gitURL string) ([]*access.WorkspaceAccess, error) {
	return []*access.WorkspaceAccess{t.workspace}, nil
}

// GetGithubAppToken returns the github app token for the installation
func (t *fakeTenantService) GetGithubAppToken(ctx context.Context, log *logrus.Entry, installationID int64) (*domain.InstallationToken, error) {
	return &domain.InstallationToken{}, nil
}
