package hook

import (
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"
)

func writeResult(log *logrus.Entry, w http.ResponseWriter, message string) {
	_, err := w.Write([]byte(message))
	if err != nil {
		log.WithError(err).Debugf("failed to write message: %s", message)
	}
}

func responseHTTPError(w http.ResponseWriter, statusCode int, message string, args ...interface{}) {
	response := fmt.Sprintf(message, args...)

	logrus.WithFields(logrus.Fields{
		"response":    response,
		"status-code": statusCode,
	}).Info(response)
	http.Error(w, response, statusCode)
}
