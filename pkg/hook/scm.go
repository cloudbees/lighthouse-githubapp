package hook

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/cloudbees/jx-tenant-service/pkg/domain"
	"github.com/cloudbees/jx-tenant-service/pkg/model"
	"github.com/cloudbees/lighthouse-githubapp/pkg/flags"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/factory"
	"github.com/jenkins-x/go-scm/scm/transport"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func defaultScmTransport(scmClient *scm.Client) {
	if scmClient.Client == nil {
		scmClient.Client = http.DefaultClient
	}
	if scmClient.Client.Transport == nil {
		scmClient.Client.Transport = http.DefaultTransport
	}
}

func (o *HookOptions) getInstallScmClient(log *logrus.Entry, ctx context.Context, ref *scm.InstallationRef) (*scm.Client, *domain.InstallationToken, error) {
	if ref == nil || ref.ID == 0 {
		return nil, nil, fmt.Errorf("missing installation.ID on webhok")
	}
	key := model.Int64ToA(ref.ID)
	item, ok := o.tokenCache.Get(key)
	if ok {
		tokenResource, ok := item.(*domain.InstallationToken)
		if ok && tokenResource != nil {
			token := tokenResource.Token
			if token != "" {
				expires := tokenResource.ExpiresAt
				if expires == nil || time.Now().Before(expires.Add(tokenCacheExpireDelta)) {
					scmClient, err := o.createInstallScmClient(log, ctx, tokenResource)
					return scmClient, tokenResource, err
				}
			}
		}
	}

	tokenResource, err := o.tenantService.GetGithubAppToken(log, ref.ID)
	if err != nil {
		return nil, tokenResource, errors.Wrapf(err, "failed to get the GitHub App token for installation %s", key)
	}
	if tokenResource != nil && tokenResource.Token != "" {
		duration := tokenCacheExpiration
		if tokenResource.ExpiresAt != nil {
			duration = tokenResource.ExpiresAt.Sub(time.Now())
		}
		o.tokenCache.Set(key, tokenResource, duration)
	}
	scmClient, err := o.createInstallScmClient(log, ctx, tokenResource)
	if err != nil {
		return scmClient, tokenResource, err
	}
	return scmClient, tokenResource, err
}

func (o *HookOptions) createInstallScmClient(log *logrus.Entry, ctx context.Context, tokenResource *domain.InstallationToken) (*scm.Client, error) {
	if tokenResource == nil {
		return nil, fmt.Errorf("no GitHub App token returned")
	}
	token := tokenResource.Token
	if token == "" {
		return nil, fmt.Errorf("empty GitHub App token returned")
	}
	client, _, _, err := o.createSCMClient(token)
	if err != nil {
		log.WithError(err).Error("failed to create real SCM client")
		return nil, err
	}
	return client, nil
}

func (o *HookOptions) createSCMClient(token string) (*scm.Client, string, string, error) {
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

// creates a client for using go-scm using the App's ID and private key
func (o *HookOptions) createAppsScmClient() (*scm.Client, int, error) {
	privateKeyFile := flags.AppPrivateKeyFile.Value()
	if privateKeyFile == "" {
		logrus.Fatalf("missing private key file environment variable $GITHUB_APP_PRIVATE_KEY_FILE")
	}
	appID := flags.GitHubAppID.Value()
	if appID == 0 {
		logrus.Fatalf("missing private key file environment variable GITHUB_APP_ID")
	}
	kind := flags.GitKind.Value()
	serverURL := flags.GitServer.Value()
	scmClient, err := factory.NewClient(kind, serverURL, "")
	if err != nil {
		return scmClient, appID, err
	}

	// add Apps installation token
	defaultScmTransport(scmClient)
	if err != nil {
		return nil, appID, errors.Wrapf(err, "failed to create SCM transport")
	}
	logrus.Infof("using GitHub App ID %d", appID)
	tr, err := ghinstallation.NewAppsTransportKeyFromFile(scmClient.Client.Transport, appID, privateKeyFile)
	if err != nil {
		return nil, appID, errors.Wrapf(err, "failed to create the Apps transport for AppID %v and file %s", appID, privateKeyFile)
	}
	scmClient.Client.Transport = tr
	return scmClient, appID, err
}
