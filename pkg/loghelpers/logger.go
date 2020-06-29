package loghelpers

import (
	"github.com/cloudbees/lighthouse-githubapp/pkg/version"
	"github.com/jenkins-x/jx-logging/pkg/log"
	stackdriver "github.com/jenkins-x/logrus-stackdriver-formatter/pkg/stackdriver"
	"github.com/sirupsen/logrus"
)

// InitLogrus initialises logging nicely
func InitLogrus() {
	// lets force jx to initialise
	log.Logger()

	formatter := stackdriver.NewFormatter(
		stackdriver.WithService("lighthouse-githubapp"),
		stackdriver.WithVersion(*version.GetBuildVersion()),
	)

	logrus.SetFormatter(formatter)

}
