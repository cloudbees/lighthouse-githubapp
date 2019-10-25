package connectors

import (
	"github.com/heptio/sonobuoy/pkg/dynamic"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/jxfactory"
	"github.com/jenkins-x/jx/pkg/kube"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	// this is so that we load the auth plugins so we can connect to, say, GCP

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

type CachingFactory struct {
	factory             jxfactory.Factory
	ns                  string
	apiExtensionsClient apiextensionsclientset.Interface
	kubeClient          kubernetes.Interface
	jxClient            versioned.Interface
	tektonClient        tektonclient.Interface
	dynamicClient       *dynamic.APIHelper
}

// NewCachingFactory creates a new client factory which caches clients across invocations
func NewCachingFactory(factory jxfactory.Factory, ns string) jxfactory.Factory {
	return &CachingFactory{factory: factory, ns: ns}
}

func (f *CachingFactory) WithBearerToken(token string) jxfactory.Factory {
	return f.factory.WithBearerToken(token)
}

func (f *CachingFactory) ImpersonateUser(user string) jxfactory.Factory {
	return f.factory.ImpersonateUser(user)
}

func (f *CachingFactory) CreateKubeClient() (kubernetes.Interface, string, error) {
	var err error
	if f.kubeClient == nil {
		f.kubeClient, _, err = f.factory.CreateKubeClient()
	}
	return f.kubeClient, f.ns, err
}

func (f *CachingFactory) CreateKubeConfig() (*rest.Config, error) {
	return f.factory.CreateKubeConfig()
}

func (f *CachingFactory) CreateJXClient() (versioned.Interface, string, error) {
	var err error
	if f.jxClient == nil {
		f.jxClient, _, err = f.factory.CreateJXClient()
	}
	return f.jxClient, f.ns, err
}

func (f *CachingFactory) CreateTektonClient() (tektonclient.Interface, string, error) {
	var err error
	if f.tektonClient == nil {
		f.tektonClient, _, err = f.factory.CreateTektonClient()
	}
	return f.tektonClient, f.ns, err
}

func (f *CachingFactory) KubeConfig() kube.Kuber {
	return f.factory.KubeConfig()
}
