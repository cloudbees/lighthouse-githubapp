#!/bin/bash
jx promote -b -e raccoon --timeout 1h --version ${VERSION} --helm-repo-url=https://storage.googleapis.com/chartmuseum.jenkins-x.io
