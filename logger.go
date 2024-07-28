package uker

import (
	"encoding/json"
	"fmt"

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
	LogFormatter        logrus.Formatter
	FluentPostTag       string
	FluentConfiguration *fluent.Config
}

func NewLogger(c LoggerConfig) (*logrus.Logger, error) {
	log := logrus.New()

	// Create a Fluentd logger
	fluentInstance, err := fluent.New(*c.FluentConfiguration)

	if err != nil {
		return nil, fmt.Errorf("failed to connect to Fluentd: %v", err)
	}

	defer fluentInstance.Close()

	// Create a FluentdWriter
	fluentWriter := &fluentdWriter{
		tag:    c.FluentPostTag,
		logger: fluentInstance,
	}

	// Set up logrus to use the FluentdWriter
	log.SetFormatter(c.LogFormatter)
	log.SetOutput(fluentWriter)

	return log, nil
}
