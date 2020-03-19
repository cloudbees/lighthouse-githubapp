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

type Scm struct{}

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
		return nil, nil, fmt.Errorf("missing installation.ID on webhook")
	}
	log.Infof("getInstallScmClient for install %d", ref.ID)
	key := model.Int64ToA(ref.ID)
	item, ok := o.tokenCache.Get(key)
	if ok {
		tokenResource, ok := item.(*domain.InstallationToken)
		log.Infof("found token in cache")
		if ok && tokenResource != nil {
			token := tokenResource.Token
			if token != "" {
				expires := tokenResource.ExpiresAt
				if expires == nil || time.Now().Before(expires.Add(tokenCacheExpireDelta)) {
					log.Infof("token is not expired")
					scmClient, err := o.createInstallScmClient(log, ctx, tokenResource)
					return scmClient, tokenResource, err
				} else {
					log.Infof("token is may be expired")
				}
			}
		}
	}

	log.Infof("requesting new github app token for install %d", ref.ID)
	tokenResource, err := o.tenantService.GetGithubAppToken(ctx, log, ref.ID)
	if err != nil {
		return nil, tokenResource, errors.Wrapf(err, "failed to get the GitHub App token for installation %s", key)
	}
	if tokenResource != nil && tokenResource.Token != "" {
		duration := tokenCacheExpiration
		if tokenResource.ExpiresAt != nil {
			// lets use the lowest duration
			expireDuration := tokenResource.ExpiresAt.Sub(time.Now()) - tokenCacheExpireDelta
			if expireDuration < duration {
				duration = expireDuration
			}
		}
		if duration > 0 {
			log.Infof("storing token in cache for install %d", ref.ID)
			o.tokenCache.Set(key, tokenResource, duration)
		}
	}
	scmClient, err := o.createInstallScmClient(log, ctx, tokenResource)
	if err != nil {
		log.Errorf("error calling createInstallScmClient: %s", err)
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
func createAppsScmClient() (*scm.Client, int, error) {
	logrus.Debugf("createAppsScmClient")
	privateKeyFile := flags.AppPrivateKeyFile.Value()
	if privateKeyFile == "" {
		logrus.Errorf("missing private key file environment variable LHA_PRIVATE_KEY_FILE")
		return nil, 0, errors.New("Missing Github APP Private key")
	}
	appID := flags.GitHubAppID.Value()
	if appID == 0 {
		logrus.Errorf("missing environment variable LHA_APP_ID")
		return nil, 0, errors.New("Missing Github APP ID")
	}
	kind := flags.GitKind.Value()
	serverURL := flags.GitServer.Value()
	scmClient, err := factory.NewClient(kind, serverURL, "")
	if err != nil {
		logrus.Errorf("failed to create scm apps client %v", err)
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
		logrus.Errorf("failed to create transport %v", err)
		return nil, appID, errors.Wrapf(err, "failed to create the Apps transport for AppID %v and file %s", appID, privateKeyFile)
	}
	scmClient.Client.Transport = tr
	return scmClient, appID, err
}
