module github.com/cloudbees/lighthouse-githubapp

go 1.13

require (
	github.com/blang/semver v3.5.1+incompatible
	github.com/bradleyfalzon/ghinstallation v0.1.2
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/cloudbees/jx-tenant-service v0.0.777
	github.com/gofrs/uuid v3.3.0+incompatible // indirect
	github.com/gorilla/mux v1.7.3
	github.com/jenkins-x/go-scm v1.5.145
	github.com/jenkins-x/jx-logging v0.0.10
	github.com/jenkins-x/logrus-stackdriver-formatter v0.2.3
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.6.0
	github.com/stretchr/testify v1.6.1
	gopkg.in/DataDog/dd-trace-go.v1 v1.25.0
)

exclude github.com/jenkins-x/jx/pkg/prow v0.0.0-20191018175829-4badc08866cd

replace github.com/heptio/sonobuoy => github.com/jenkins-x/sonobuoy v0.11.7-0.20190318120422-253758214767

replace k8s.io/api => k8s.io/api v0.16.5

replace k8s.io/metrics => k8s.io/metrics v0.0.0-20190819143841-305e1cef1ab1

replace k8s.io/apimachinery => k8s.io/apimachinery v0.16.5

replace k8s.io/client-go => k8s.io/client-go v0.16.5

replace k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190819143637-0dbe462fe92d

replace github.com/Azure/azure-sdk-for-go => github.com/Azure/azure-sdk-for-go v23.2.0+incompatible

replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.1+incompatible

replace github.com/banzaicloud/bank-vaults => github.com/banzaicloud/bank-vaults v0.0.0-20191212164220-b327d7f2b681

replace github.com/banzaicloud/bank-vaults/pkg/sdk => github.com/banzaicloud/bank-vaults/pkg/sdk v0.0.0-20191212164220-b327d7f2b681

replace github.com/TV4/logrus-stackdriver-formatter => github.com/jenkins-x/logrus-stackdriver-formatter v0.1.1-0.20200408213659-1dcf20c371bb

replace k8s.io/test-infra => github.com/jenkins-x/test-infra v0.0.0-20200611142252-211a92405c22

replace gomodules.xyz/jsonpatch/v2 => gomodules.xyz/jsonpatch/v2 v2.0.1
