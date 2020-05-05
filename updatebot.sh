#!/bin/bash

ENVS="raccoon arcalos-staging-mgmt arcalos-prod-mgmt"
for ENV in $ENVS; do
	jx promote -b -e ${ENV} --no-poll --no-wait --timeout 1h --version ${VERSION} --helm-repo-url=https://storage.googleapis.com/chartmuseum.jenkins-x.io
done

exit 0
