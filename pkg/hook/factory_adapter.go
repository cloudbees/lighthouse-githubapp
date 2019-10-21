package hook

import (
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
}

func ToJXFactory(factory *connector.ConfigClientFactory) jxfactory.Factory {
	return &factoryAdapter{factory}
}

func (f *factoryAdapter) WithBearerToken(token string) jxfactory.Factory {
	return f
}

func (f *factoryAdapter) ImpersonateUser(user string) jxfactory.Factory {
	return f
}

func (f *factoryAdapter) CreateKubeClient() (kubernetes.Interface, string, error) {
	return f.CreateKubeClient()
}

func (f *factoryAdapter) CreateKubeConfig() (*rest.Config, error) {
	return f.CreateKubeConfig()
}

func (f *factoryAdapter) CreateJXClient() (versioned.Interface, string, error) {
	return f.CreateJXClient()
}

func (f *factoryAdapter) CreateTektonClient() (tektonclient.Interface, string, error) {
	return f.CreateTektonClient()
}

func (f *factoryAdapter) KubeConfig() kube.Kuber {
	return f.KubeConfig()
}
