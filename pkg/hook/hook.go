package hook

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/cloudbees/lighthouse-githubapp/pkg/hmac"
	"github.com/cloudbees/lighthouse-githubapp/pkg/tenant"

	"github.com/cloudbees/lighthouse-githubapp/pkg/util"

	"github.com/cloudbees/jx-tenant-service/pkg/gcloudhelpers"
	"github.com/cloudbees/lighthouse-githubapp/pkg/flags"
	"github.com/cloudbees/lighthouse-githubapp/pkg/hook/connectors"
	"github.com/cloudbees/lighthouse-githubapp/pkg/schedulers"
	"github.com/jenkins-x/go-scm/scm"
	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/jxfactory/connector"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/lighthouse/pkg/plumber"
	"github.com/jenkins-x/lighthouse/pkg/prow/config"
	"github.com/jenkins-x/lighthouse/pkg/prow/git"
	lhhook "github.com/jenkins-x/lighthouse/pkg/prow/hook"
	"github.com/jenkins-x/lighthouse/pkg/prow/plugins"
	lhwebhook "github.com/jenkins-x/lighthouse/pkg/webhook"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	muxtrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/gorilla/mux"
)

const (
	repoNotConfiguredMessage = "repository not configured"
)

type HookOptions struct {
	Port          string
	Path          string
	Version       string
	tokenCache    *cache.Cache
	tenantService tenant.TenantService
	githubApp     *GithubApp
	secretFn      func(webhook scm.Webhook) (string, error)
	client        *http.Client
}

// NewHook create a new hook handler
func NewHook() (*HookOptions, error) {
	tokenCache := cache.New(tokenCacheExpiration, tokenCacheExpiration)
	tenantService := tenant.NewTenantService("")
	githubApp, err := NewGithubApp()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create hook")
	}

	secretFn := func(webhook scm.Webhook) (string, error) {
		return flags.HmacToken.Value(), nil
	}

	return &HookOptions{
		Path:          HookPath,
		Port:          flags.HttpPort.Value(),
		Version:       "TODO",
		tokenCache:    tokenCache,
		tenantService: tenantService,
		githubApp:     githubApp,
		secretFn:      secretFn,
	}, nil
}

func (o *HookOptions) Handle(mux *muxtrace.Router) {
	mux.Handle(GithubAppPath, http.HandlerFunc(o.githubApp.handleInstalledRequests))
	mux.Handle(TestTokenPath, http.HandlerFunc(o.handleTokenValid))
	mux.Handle(HealthPath, http.HandlerFunc(o.health))
	mux.Handle(ReadyPath, http.HandlerFunc(o.ready))
	mux.Handle(SetupPath, http.HandlerFunc(o.setup))

	mux.Handle("/", http.HandlerFunc(o.defaultHandler))
	mux.Handle(o.Path, http.HandlerFunc(o.handleWebHookRequests))
}

// health returns either HTTP 204 if the service is healthy, otherwise nothing ('cos it's dead).
func (o *HookOptions) health(w http.ResponseWriter, r *http.Request) {
	util.TraceLogger(r.Context()).Info("Health check")
	w.WriteHeader(http.StatusNoContent)
}

// ready returns either HTTP 204 if the service is ready to serve requests, otherwise HTTP 503.
func (o *HookOptions) ready(w http.ResponseWriter, r *http.Request) {
	util.TraceLogger(r.Context()).Debug("Ready check")
	if o.isReady() {
		w.WriteHeader(http.StatusNoContent)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
}

// setup handle the setup URL
func (o *HookOptions) setup(w http.ResponseWriter, r *http.Request) {
	util.TraceLogger(r.Context()).Debug("setup")

	action := r.URL.Query().Get("setup_action")
	installationID := r.URL.Query().Get("installation_id")
	message := fmt.Sprintf(`Welcome to the Jenkins X Bot for installation: %s action: %s`, installationID, action)

	_, err := w.Write([]byte(message))
	if err != nil {
		util.TraceLogger(r.Context()).Debugf("failed to write the setup: %v", err)
	}
}

func (o *HookOptions) defaultHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == o.Path || strings.HasPrefix(path, o.Path+"/") {
		o.handleWebHookRequests(w, r)
		return
	}
	path = strings.TrimPrefix(path, "/")
	if path == "" || path == "index.html" {
		o.getIndex(w, r)
		return
	}
	http.Error(w, fmt.Sprintf("unknown path %s", path), 404)
}

// getIndex returns a simple home page
func (o *HookOptions) getIndex(w http.ResponseWriter, r *http.Request) {
	l := util.TraceLogger(r.Context())
	l.Debug("GET index")
	message := fmt.Sprintf(`Hello from Jenkins X Lighthouse version: %s

For more information see: https://github.com/jenkins-x/lighthouse
`, o.Version)

	_, err := w.Write([]byte(message))
	if err != nil {
		l.Debugf("failed to write the index: %v", err)
	}
}

func (o *HookOptions) isReady() bool {
	// TODO a better readiness check
	return true
}

func (o *HookOptions) onInstallHook(ctx context.Context, log *logrus.Entry, hook *scm.InstallationHook) error {
	install := hook.Installation
	id := install.ID
	fields := map[string]interface{}{
		"Action":         hook.Action.String(),
		"InstallationID": id,
		"Function":       "onInstallHook",
	}
	log = log.WithFields(fields)
	log.Infof("installHook %+v", hook)
	log.Infof("installHook Installation - %+v", hook.Installation)
	log.Infof("installHook Repos - %+v", hook.Repos)

	// ets register / unregister repositories to the InstallationID
	if hook.Action == scm.ActionCreate {
		ownerURL := hook.Installation.Account.Link
		log = log.WithField("Owner", ownerURL)
		if ownerURL == "" {
			err := fmt.Errorf("missing ownerURL on install webhook")
			log.Error(err.Error())
			return err
		}

		return o.tenantService.AppInstall(ctx, log, id, ownerURL)
	} else if hook.Action == scm.ActionDelete {
		return o.tenantService.AppUnnstall(ctx, log, id)
	} else {
		log.Warnf("ignore unknown action")
		return nil
	}
}

func (o *HookOptions) onGeneralHook(ctx context.Context, log *logrus.Entry, install *scm.InstallationRef, webhook scm.Webhook, githubDeliveryEvent string, bodyBytes []byte) error {
	id := install.ID
	repo := webhook.Repository()

	// TODO this should be fixed in go-scm
	if repo.FullName == "" {
		log.Debugf("repo.FullName is empty, calculating full name as '%s/%s'", repo.Namespace, repo.Name)
		repo.FullName = repo.Namespace + "/" + repo.Name
	}
	fields := map[string]interface{}{
		"InstallationID": id,
		"FullName":       repo.FullName,
		"Link":           repo.Link,
	}
	log = log.WithFields(fields)
	u := repo.Link
	if u == "" {
		log.Warnf("ignoring webhook as no repository URL for '%s'", repo.FullName)
		return nil
	}

	log.Infof("onGeneralHook - %+v", webhook)
	workspaces, err := o.tenantService.FindWorkspaces(ctx, log, id, u)
	if err != nil {
		log.WithError(err).Errorf("Unable to find workspaces for %s", repo.FullName)
		return err
	}

	for _, ws := range workspaces {
		log := log.WithFields(ws.LogFields())
		log.Infof("notifying workspace %s for %s", ws.Project, repo.FullName)

		if ws.LighthouseURL != "" && ws.HMAC != "" {
			log.Infof("invoking webhook relay here! %s with hmac %s", ws.LighthouseURL, ws.HMAC)
			log.Infof("relaying %s", string(bodyBytes))

			decodedHmac, err := base64.StdEncoding.DecodeString(ws.HMAC)
			if err != nil {
				log.WithError(err).Errorf("unable to decode hmac")
				continue
			}

			g := hmac.NewGenerator(decodedHmac)
			signature := g.HubSignature(bodyBytes)

			if o.client == nil {
				o.client = &http.Client{}
			}
			req, err := http.NewRequest("POST", ws.LighthouseURL, bytes.NewReader(bodyBytes))
			req.Header.Add("X-GitHub-Event", string(webhook.Kind()))
			req.Header.Add("X-GitHub-Delivery", githubDeliveryEvent)
			req.Header.Add("X-Hub-Signature", signature)

			err = o.retryWebhookDelivery(req, log)
			if err != nil {
				log.WithError(err).Error("failed to deliver webhook")
				continue
			}
		} else {
			kubeConfig := ws.KubeConfig
			if kubeConfig == "" {
				log.Error("no KubeConfig for workspace")
				continue
			}
			f, err := gcloudhelpers.CreateFactoryFromKubeConfig(kubeConfig)
			if err != nil {
				log.WithError(err).Error("failed to create remote client factory")
				continue
			}

			// lets parse the Scheduler json
			jsonText := ws.JSON
			if jsonText == "" {
				log.Errorf("no Scheduler JSON for workspace %s and repo %s", ws.Project, repo.FullName)
				continue
			}
			scheduler := &v1.Scheduler{}
			err = json.Unmarshal([]byte(jsonText), scheduler)
			if err != nil {
				log.WithError(err).Errorf("failed to parse Scheduler JSON for workspace %s and repo %s", ws.Project, repo.FullName)
				continue
			}

			log.WithField("Scheduler", scheduler.Name)
			log.WithField("Bot", flags.BotName.Value())

			err = o.invokeLighthouse(log, webhook, f, ws.Namespace, scheduler, install)
			if err != nil {
				log.WithError(err).Errorf("failed to invoke remote lighthouse for workspace %s and repo %s", ws.Project, repo.FullName)
				return err
			}
		}
	}

	log.Infof("%d workspaces interested in repository %s", len(workspaces), repo.FullName)

	if len(workspaces) == 0 {
		return errors.Errorf("no workspaces interested in repository '%s', backing off...", repo.FullName)
	}

	return nil
}

func (o *HookOptions) invokeRemoteLighthouseViaProxy(log *logrus.Entry, webhook scm.Webhook, factory *connector.ConfigClientFactory, ns string) error {
	log.Info("invoking remote lighthouse")
	kubeClient, err := factory.CreateKubeClient()
	if err != nil {
		return errors.Wrap(err, "failed to create remote kubeClient")
	}
	params := map[string]string{}
	data, err := kubeClient.CoreV1().Services(ns).ProxyGet("http", "hook", "80", "/", params).DoRaw()
	if err != nil {
		return errors.Wrap(err, "failed to get hook")
	}
	log.Infof("got response from hook: %s", string(data))
	return nil
}

func (o *HookOptions) invokeLighthouse(log *logrus.Entry, webhook scm.Webhook, f *connector.ConfigClientFactory, ns string, scheduler *v1.Scheduler, installRef *scm.InstallationRef) error {
	log.Info("invoking lighthouse")

	clientFactory := connectors.ToJXFactory(f, ns)
	kubeClient, _, err := clientFactory.CreateKubeClient()
	if err != nil {
		err = errors.Wrap(err, "failed to create KubeClient")
		log.Error(err.Error())
		return err
	}
	jxClient, _, err := clientFactory.CreateJXClient()
	if err != nil {
		err = errors.Wrap(err, "failed to create JXClient")
		log.Error(err.Error())
		return err
	}

	serverURL := webhook.Repository().Link

	gitClient, _ := git.NewClient(serverURL, flags.GitKind.Value())

	ctx := context.Background()
	scmClient, tokenResource, err := o.getInstallScmClient(log, ctx, installRef)

	if webhook.Kind() == scm.WebhookKindPullRequest {
		o.verifyScmClient(log, scmClient, webhook.Repository())
	}

	botUser := flags.BotName.Value()
	gitClient.SetCredentials(botUser, func() []byte {
		return []byte(tokenResource.Token)
	})

	metapipelineClient, err := plumber.NewMetaPipelineClient(clientFactory)
	if err != nil {
		err = errors.Wrap(err, "failed to create Metapipeline Client")
		log.Error(err.Error())
		return err
	}

	server := &lhhook.Server{
		ClientAgent: &plugins.ClientAgent{
			BotName:            botUser,
			SCMProviderClient:  scmClient,
			KubernetesClient:   kubeClient,
			GitClient:          gitClient,
			MetapipelineClient: metapipelineClient,
		},
		ClientFactory:      clientFactory,
		MetapipelineClient: metapipelineClient,
		Plugins:            &plugins.ConfigAgent{},
		ConfigAgent:        &config.Agent{},
	}
	err = o.updateProwConfiguration(log, webhook, server, scheduler, jxClient, ns)
	if err != nil {
		err = errors.Wrap(err, "failed to update prow configuration")
		log.Error(err.Error())
		return err
	}

	localHook := lhwebhook.NewWebhook(clientFactory, server)

	log.Info("about to invoke lighthouse")

	l, output, err := localHook.ProcessWebHook(log, webhook)
	if err != nil {
		err = errors.Wrap(err, "failed to process webhook")
		l.WithError(err).Error(err.Error())
		return err
	}
	l.Infof(output)
	return nil
}

func (o *HookOptions) updateProwConfiguration(log *logrus.Entry, webhook scm.Webhook, server *lhhook.Server, scheduler *v1.Scheduler, jxClient versioned.Interface, ns string) error {
	devEnv := kube.CreateDefaultDevEnvironment(ns)
	fn := func(versioned.Interface, string) (map[string]*v1.Scheduler, *v1.SourceRepositoryGroupList, *v1.SourceRepositoryList, error) {
		m := map[string]*v1.Scheduler{
			scheduler.Name: scheduler,
		}
		repo := webhook.Repository()
		sr := v1.SourceRepository{
			Spec: v1.SourceRepositorySpec{
				URL:  repo.Link,
				Org:  repo.Namespace,
				Repo: repo.Name,
			},
		}
		sr.Spec.Scheduler.Name = scheduler.Name
		srgs := &v1.SourceRepositoryGroupList{}
		srs := &v1.SourceRepositoryList{
			Items: []v1.SourceRepository{
				sr,
			},
		}
		return m, srgs, srs, nil
	}

	prowConfig, prowPlugins, err := schedulers.GenerateProw(false, false, jxClient, ns, "default", devEnv, fn)
	if err != nil {
		err = errors.Wrap(err, "failed to generate prow config")
		log.WithError(err)
		return err
	}
	server.ConfigAgent.Set(prowConfig)
	server.Plugins.Set(prowPlugins)
	return nil
}

// verifyScmClient lets try verify the scm client on a webhook
func (o *HookOptions) verifyScmClient(log *logrus.Entry, scmClient *scm.Client, repository scm.Repository) {
	log = log.WithField("TestPR", fmt.Sprintf("%s #%d", verifyRepository, verifyPullRequest))
	ctx := context.Background()
	labels, _, err := scmClient.PullRequests.ListLabels(ctx, verifyRepository, verifyPullRequest, scm.ListOptions{Size: 100})
	if err != nil {
		log.WithError(err).Error("failed to list labels on PR")
		return
	}
	buffer := strings.Builder{}
	buffer.WriteString("Found labels: ")
	for _, label := range labels {
		buffer.WriteString(" ")
		buffer.WriteString(label.Name)
	}
	log.Info(buffer.String())
}

// retryWebhookDelivery attempts to deliver the relayed webhook, but will retry a few times if the response is a 500 with
// "repository not configured" in the body, in case the remote Lighthouse doesn't yet have this repository in its configuration.
func (o *HookOptions) retryWebhookDelivery(req *http.Request, log *logrus.Entry) error {
	f := func() error {
		resp, err := o.client.Do(req)
		if err != nil {
			return backoff.Permanent(err)
		}
		// If we got a 500, check if it's got the "repository not configured" string in the body. If so, we retry.
		if resp.StatusCode == 500 {
			respBody, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return backoff.Permanent(errors.Wrap(err, "parsing response body"))
			}
			resp.Body.Close()
			if strings.Contains(string(respBody), repoNotConfiguredMessage) {
				return errors.New("repository not configured in Lighthouse")
			}
		}

		// If we got anything other than a 2xx, error permanently.
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return backoff.Permanent(errors.Errorf("%s not available, error was %d %s", req.URL.String(), resp.StatusCode, resp.Status))
		}

		// And finally, if we haven't gotten any errors, just return nil because we're good.
		log.Infof("got response %+v", resp)
		return nil
	}
	bo := backoff.NewExponentialBackOff()
	// Try again after 1/2/4/8 seconds if necessary, for up to 30 seconds.
	bo.InitialInterval = 1 * time.Second
	bo.MaxElapsedTime = 30 * time.Second
	bo.Reset()
	return backoff.Retry(f, bo)
}
