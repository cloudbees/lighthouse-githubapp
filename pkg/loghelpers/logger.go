package loghelpers

import (
	stackdriver "github.com/TV4/logrus-stackdriver-formatter"
	"github.com/cloudbees/lighthouse-githubapp/pkg/version"
	jxlogger "github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/sirupsen/logrus"
)

// InitLogrus initialises logging nicely
func InitLogrus() {
	// lets force jx to initialise
	log.Logger()
	jxlogger.Logger()

	formatter := stackdriver.NewFormatter(
		stackdriver.WithService("lighthouse-githubapp"),
		stackdriver.WithVersion(*version.GetBuildVersion()),
	)

	logrus.SetFormatter(formatter)

}
