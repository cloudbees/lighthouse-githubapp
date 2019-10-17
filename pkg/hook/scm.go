package hook

import (
	"context"
	"fmt"
	"net/http"

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

func (o *HookOptions) getInstallScmClient(log *logrus.Entry, ctx context.Context, ref *scm.InstallationRef) (*scm.Client, error) {
	// TODO use cache...
	if ref == nil || ref.ID == 0 {
		return nil, fmt.Errorf("missing installation.ID on webhok")
	}
	appsClient, _, err := o.createAppsScmClient()
	if err != nil {
		return nil, err
	}
	tokenResource, _, err := appsClient.Apps.CreateInstallationToken(ctx, ref.ID)
	if err != nil {
		log.WithError(err).Error("failed to create installation token")
		return nil, err
	}
	if tokenResource == nil {
		return nil, fmt.Errorf("no token returned from CreateInstallationToken()")
	}
	token := tokenResource.Token
	if token == "" {
		return nil, fmt.Errorf("empty token returned from CreateInstallationToken()")
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
