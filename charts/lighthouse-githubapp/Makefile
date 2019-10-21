CHART_REPO := http://jenkins-x-chartmuseum:8080
CURRENT=$(pwd)
NAME := lighthouse-githubapp
OS := $(shell uname)
RELEASE_VERSION := $(shell cat ../../VERSION)

build: clean
	rm -rf requirements.lock
	sed -i -e "s/version:.*/version: $(VERSION)/" Chart.yaml
	sed -i -e "s/imagetag: <VERSION>/imagetag: $(VERSION)/" values.yaml
	jx step helm build
	helm lint
	helm template .

install: clean build
	helm install . --name ${NAME}

upgrade: clean build
	helm upgrade ${NAME} .

delete:
	helm delete --purge ${NAME}

clean:
	rm -rf charts
	rm -rf ${NAME}*.tgz