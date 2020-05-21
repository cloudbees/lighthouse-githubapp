package hook

import (
	"net/http"

	"github.com/bradleyfalzon/ghinstallation"
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
