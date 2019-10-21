package hook

import (
	"fmt"

	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/jxfactory"
	"github.com/jenkins-x/jx/pkg/jxfactory/connector"
	"github.com/jenkins-x/jx/pkg/kube"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// TODO we should move this back into jenkins-x/jx repo!
type factoryAdapter struct {
	factory *connector.ConfigClientFactory
	ns      string
}

func ToJXFactory(factory *connector.ConfigClientFactory, ns string) jxfactory.Factory {
	return &factoryAdapter{factory, ns}
}

func (f *factoryAdapter) WithBearerToken(token string) jxfactory.Factory {
	return f
}

func (f *factoryAdapter) ImpersonateUser(user string) jxfactory.Factory {
	return f
}

func (f *factoryAdapter) CreateKubeClient() (kubernetes.Interface, string, error) {
	client, err := f.factory.CreateKubeClient()
	return client, f.ns, err
}

func (f *factoryAdapter) CreateKubeConfig() (*rest.Config, error) {
	return nil, fmt.Errorf("TODO")
}

func (f *factoryAdapter) CreateJXClient() (versioned.Interface, string, error) {
	client, err := f.factory.CreateJXClient()
	return client, f.ns, err
}

func (f *factoryAdapter) CreateTektonClient() (tektonclient.Interface, string, error) {
	client, err := f.factory.CreateTektonClient()
	return client, f.ns, err
}

func (f *factoryAdapter) KubeConfig() kube.Kuber {
	// TODO
	return nil
	//return f.factory.KubeConfig()
}
