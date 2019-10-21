package hook

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/cloudbees/lighthouse-githubapp/pkg/flags"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/factory"
	"github.com/jenkins-x/go-scm/scm/transport"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (o *HookOptions) createAppsScmClient() (*scm.Client, int, error) {
	scmClient, _, _, err := o.createSCMClient("")
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to create ScmClient")
	}
	defaultScmTransport(scmClient)
	appID := flags.GitHubAppID.Value()
	if err != nil {
		return nil, appID, fmt.Errorf("missing $LHA_APP_ID")
	}
	logrus.Infof("using GitHub App ID %d", appID)
	tr, err := ghinstallation.NewAppsTransportKeyFromFile(scmClient.Client.Transport, appID, o.PrivateKeyFile)
	if err != nil {
		return nil, appID, errors.Wrapf(err, "failed to create the Apps transport for AppID %v and file %s", appID, o.PrivateKeyFile)
	}
	scmClient.Client.Transport = tr
	return scmClient, appID, err
}

func defaultScmTransport(scmClient *scm.Client) {
	if scmClient.Client == nil {
		scmClient.Client = http.DefaultClient
	}
	if scmClient.Client.Transport == nil {
		scmClient.Client.Transport = http.DefaultTransport
	}
}

func (o *HookOptions) getInstallScmClient(log *logrus.Entry, ctx context.Context, ref *scm.InstallationRef) (*scm.Client, *scm.InstallationToken, error) {
	if ref == nil || ref.ID == 0 {
		return nil, nil, fmt.Errorf("missing installation.ID on webhok")
	}
	key := fmt.Sprintf("%v", ref.ID)
	item, ok := o.tokenCache.Get(key)
	if ok {
		tokenResource, ok := item.(*scm.InstallationToken)
		if ok && tokenResource != nil {
			token := tokenResource.Token
			if token != "" {
				expires := tokenResource.ExpiresAt
				if expires == nil || time.Now().Before(expires.Add(tokenCacheExpireDelta)) {
					scmClient, _, _, err := o.createSCMClient(token)
					return scmClient, tokenResource, err
				}
			}
		}
	}
	log.WithField("InstallationID", ref.ID).Infof("requesting new installation token")
	scmClient, tokenResource, err := o.createInstallScmClient(log, ctx, ref)
	if err != nil {
		return scmClient, tokenResource, err
	}
	if tokenResource != nil && tokenResource.Token != "" {
		duration := tokenCacheExpiration
		if tokenResource.ExpiresAt != nil {
			duration = tokenResource.ExpiresAt.Sub(time.Now())
		}
		o.tokenCache.Set(key, tokenResource, duration)
	}
	return scmClient, tokenResource, err
}

func (o *HookOptions) createInstallScmClient(log *logrus.Entry, ctx context.Context, ref *scm.InstallationRef) (*scm.Client, *scm.InstallationToken, error) {
	appsClient, _, err := o.createAppsScmClient()
	if err != nil {
		return nil, nil, err
	}
	tokenResource, _, err := appsClient.Apps.CreateInstallationToken(ctx, ref.ID)
	if err != nil {
		log.WithError(err).Error("failed to create installation token")
		return nil, tokenResource, err
	}
	if tokenResource == nil {
		return nil, tokenResource, fmt.Errorf("no token returned from CreateInstallationToken()")
	}
	token := tokenResource.Token
	if token == "" {
		return nil, tokenResource, fmt.Errorf("empty token returned from CreateInstallationToken()")
	}
	client, _, _, err := o.createSCMClient(token)
	if err != nil {
		log.WithError(err).Error("failed to create real SCM client")
		return nil, tokenResource, err
	}
	return client, tokenResource, nil
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
