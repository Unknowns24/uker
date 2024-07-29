package uker

import (
	"encoding/json"

	"github.com/fluent/fluent-logger-golang/fluent"
	"github.com/sirupsen/logrus"
)

type fluentdWriter struct {
	tag    string
	logger *fluent.Fluent
}

func (f *fluentdWriter) Write(p []byte) (n int, err error) {
	var data map[string]interface{}
	err = json.Unmarshal(p, &data)
	if err != nil {
		return 0, err
	}

	err = f.logger.Post(f.tag, data)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

type LoggerConfig struct {
	LogFormatter     logrus.Formatter
	FluentPostTag    string
	FluentConnection *fluent.Fluent
}

func NewLogger(c LoggerConfig) *logrus.Logger {
	log := logrus.New()

	// Create a FluentdWriter
	fluentWriter := &fluentdWriter{
		tag:    c.FluentPostTag,
		logger: c.FluentConnection,
	}

	// Set up logrus to use the FluentdWriter
	log.SetFormatter(c.LogFormatter)
	log.SetOutput(fluentWriter)

	return log
}
