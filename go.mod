module github.com/cloudbees/lighthouse-githubapp

require (
	github.com/bradleyfalzon/ghinstallation v0.1.2
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/cloudbees/jx-tenant-service v0.0.82
	github.com/golang/protobuf v1.3.1 // indirect
	github.com/jenkins-x/go-scm v1.5.49
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/errors v0.8.1
	github.com/shurcooL/githubv4 v0.0.0-20191006152017-6d1ea27df521 // indirect
	github.com/sirupsen/logrus v1.4.2
	github.com/stretchr/testify v1.3.0
)

replace github.com/golang/lint => golang.org/x/lint v0.0.0-20180702182130-06c8688daad7

replace github.com/heptio/sonobuoy => github.com/jenkins-x/sonobuoy v0.11.7-0.20190318120422-253758214767

replace k8s.io/api => k8s.io/api v0.0.0-20181128191700-6db15a15d2d3

replace k8s.io/metrics => k8s.io/metrics v0.0.0-20181128195641-3954d62a524d

replace k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190122181752-bebe27e40fb7

replace k8s.io/client-go => k8s.io/client-go v2.0.0-alpha.0.0.20190115164855-701b91367003+incompatible

replace k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20181128195303-1f84094d7e8e

replace git.apache.org/thrift.git => github.com/apache/thrift v0.0.0-20180902110319-2566ecd5d999

replace github.com/sirupsen/logrus => github.com/jtnord/logrus v1.4.2-0.20190423161236-606ffcaf8f5d

replace github.com/Azure/azure-sdk-for-go => github.com/Azure/azure-sdk-for-go v21.1.0+incompatible

replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v10.15.5+incompatible

replace github.com/banzaicloud/bank-vaults => github.com/banzaicloud/bank-vaults v0.0.0-20190508130850-5673d28c46bd
