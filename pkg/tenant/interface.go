package tenant

import (
	"context"

	"github.com/cloudbees/jx-tenant-service/pkg/access"
	"github.com/cloudbees/jx-tenant-service/pkg/domain"
	"github.com/sirupsen/logrus"
)

type TenantService interface {
	AppInstall(ctx context.Context, log *logrus.Entry, installationID int64, ownerURL string) error
	AppUnnstall(ctx context.Context, log *logrus.Entry, installationID int64) error
	FindWorkspaces(ctx context.Context, log *logrus.Entry, installationID int64, gitURL string) ([]*access.WorkspaceAccess, error)
	GetGithubAppToken(ctx context.Context, log *logrus.Entry, installationID int64) (*domain.InstallationToken, error)
}
