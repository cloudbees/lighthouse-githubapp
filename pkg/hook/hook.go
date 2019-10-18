package hook

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/cloudbees/lighthouse-githubapp/pkg/flags"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/jx/pkg/jxfactory/connector"
	"github.com/jenkins-x/jx/pkg/jxfactory/connector/provider"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type HookOptions struct {
	Port             string
	Path             string
	Version          string
	PrivateKeyFile   string
	tokenCache       *cache.Cache
	tenantService    *TenantService
	clusterConnector connector.Client
}

// NewHook create a new hook handler
func NewHook(privateKeyFile string) (*HookOptions, error) {
	workDir, err := ioutil.TempDir("", "jx-connectors-")
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create temp dir for jx connectors")
	}

	clusterConnector := provider.NewClient(workDir)

	tokenCache := cache.New(tokenCacheExpiration, tokenCacheExpiration)
	tenantService := NewTenantService("")

	return &HookOptions{
		Path:             HookPath,
		Port:             flags.HttpPort.Value(),
		Version:          "TODO",
		PrivateKeyFile:   privateKeyFile,
		tokenCache:       tokenCache,
		tenantService:    tenantService,
		clusterConnector: clusterConnector,
	}, nil
}

func (o *HookOptions) Handle(mux *http.ServeMux) {
	mux.Handle(HealthPath, http.HandlerFunc(o.health))
	mux.Handle(ReadyPath, http.HandlerFunc(o.ready))
	mux.Handle(SetupPath, http.HandlerFunc(o.setup))

	mux.Handle("/", http.HandlerFunc(o.defaultHandler))
	mux.Handle(o.Path, http.HandlerFunc(o.handleWebHookRequests))
}

// health returns either HTTP 204 if the service is healthy, otherwise nothing ('cos it's dead).
func (o *HookOptions) health(w http.ResponseWriter, r *http.Request) {
	logrus.Debug("Health check")
	w.WriteHeader(http.StatusNoContent)
}

// ready returns either HTTP 204 if the service is ready to serve requests, otherwise HTTP 503.
func (o *HookOptions) ready(w http.ResponseWriter, r *http.Request) {
	logrus.Debug("Ready check")
	if o.isReady() {
		w.WriteHeader(http.StatusNoContent)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
}

// setup handle the setup URL
func (o *HookOptions) setup(w http.ResponseWriter, r *http.Request) {
	logrus.Debug("setup")

	action := r.URL.Query().Get("setup_action")
	installationID := r.URL.Query().Get("installation_id")
	message := fmt.Sprintf(`Welcome to the Jenkins X Bot for installation: %s action: %s`, installationID, action)

	_, err := w.Write([]byte(message))
	if err != nil {
		logrus.Debugf("failed to write the setup: %v", err)
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
	logrus.Debug("GET index")
	message := fmt.Sprintf(`Hello from Jenkins X Lighthouse version: %s

For more information see: https://github.com/jenkins-x/lighthouse
`, o.Version)

	_, err := w.Write([]byte(message))
	if err != nil {
		logrus.Debugf("failed to write the index: %v", err)
	}
}

func (o *HookOptions) isReady() bool {
	// TODO a better readiness check
	return true
}

func (o *HookOptions) onInstallHook(log *logrus.Entry, hook *scm.InstallationHook) error {
	install := hook.Installation
	id := install.ID
	fields := map[string]interface{}{
		"Action":         hook.Action.String(),
		"InstallationID": id,
		"Function":       "onInstallHook",
	}
	log = log.WithFields(fields)
	log.Infof("installHook")

	// ets register / unregister repositories to the InstallationID
	if hook.Action == scm.ActionCreate {
		ownerURL := hook.Installation.Link
		log = log.WithField("Owner", ownerURL)
		if ownerURL == "" {
			err := fmt.Errorf("missing ownerURL on install webhook")
			log.Error(err.Error())
			return err
		}

		/*
			repos := []RepositoryInfo{}
			for _, repo := range hook.Repos {
				link := strings.TrimSuffix(repo.Link, "/")
				link = strings.TrimSuffix(link, ".git")
				if link == "" {
					full := repo.FullName
					if full != "" {
						link = util.UrlJoin("https://github.com", full)
					}
				}
				if link != "" {
					repos = append(repos, RepositoryInfo{URL: link})
				}
			}
			if len(repos) == 0 {

			}
		*/
		return o.tenantService.AppInstall(log, id, ownerURL)
	} else if hook.Action == scm.ActionDelete {
		return o.tenantService.AppUnnstall(log, id)
	} else {
		log.Warnf("ignore unknown action")
		return nil
	}
}

func (o *HookOptions) onGeneralHook(log *logrus.Entry, install *scm.InstallationRef, webhook scm.Webhook) error {
	id := install.ID
	repo := webhook.Repository()
	fields := map[string]interface{}{
		"InstallationID": id,
		"Fullname":       repo.FullName,
	}
	log = log.WithFields(fields)
	u := repo.Link
	if u == "" {
		log.Warnf("ignoring webhook as no repository URL for")
		return nil
	}
	log.Infof("onGeneralHook")
	workspaces, err := o.tenantService.FindWorkspaces(log, id, u)
	if err != nil {
		return err
	}

	for _, ws := range workspaces {
		log.WithFields(ws.LogFields()).Infof("got workspace")

		// TODO we could cache these factory/clients to speed things up
		rc := &connector.RemoteConnector{GKE: &connector.GKEConnector{
			Project: ws.Project,
			Cluster: ws.Cluster,
			Region:  ws.Region,
			Zone:    ws.Zone,
		}}
		config, err := o.clusterConnector.Connect(rc)
		if err != nil {
			log.WithError(err).Error("failed to create rest config")
			return err
		}
		f := connector.NewConfigClientFactory(ws.Project, config)

		err = o.invokeRemoteLighthouse(log, webhook, f, ws.Namespace)
		if err != nil {
			log.WithError(err).Error("failed to invoke remote lighthouse")
			return err
		}
	}
	if len(workspaces) == 0 {
		log.Warnf("no workspaces interested in repository")
	}
	return nil
}

func (o *HookOptions) invokeRemoteLighthouse(log *logrus.Entry, webhook scm.Webhook, factory *connector.ConfigClientFactory, ns string) error {
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
