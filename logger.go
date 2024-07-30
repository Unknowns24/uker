package uker

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/fluent/fluent-logger-golang/fluent"
	"github.com/sirupsen/logrus"
)

type UkerFluentMetadata struct {
	Tag         string
	Source      string
	ServiceName string
	Application string
}

type UkerLogger struct {
	config *LoggerConfig
	writer *fluentdWriter
	Logger *logrus.Logger
}

type LoggerConfig struct {
	FluentMetadata     UkerFluentMetadata
	FluentConfig       fluent.Config
	LogFormatter       logrus.Formatter
	LogOnConsole       bool
	TestConnectionTime time.Duration
}

type fluentdWriter struct {
	conn   *fluent.Fluent
	config *LoggerConfig
}

func (fw *fluentdWriter) Write(p []byte) (n int, err error) {
	var data map[string]interface{}
	err = json.Unmarshal(p, &data)
	if err != nil {
		return 0, err
	}

	// Adding metadata fields if not empty inside data struct
	metadataFields := map[string]string{"application": fw.config.FluentMetadata.Application, "servicename": fw.config.FluentMetadata.ServiceName, "source": fw.config.FluentMetadata.Source}
	for label, value := range metadataFields {
		if value != "" {
			data[label] = value
		}
	}

	if fw.config.LogOnConsole {
		fmt.Println(data["msg"])
	}

	err = fw.conn.Post(fw.config.FluentMetadata.Tag, data)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func NewLogger(c *LoggerConfig) *UkerLogger {
	log := logrus.New()

	// Create a Fluentd logger
	fluentInstance, err := fluent.New(c.FluentConfig)

	if err != nil {
		panic(fmt.Sprintf("failed to connect to Fluentd: %v", err))
	}

	// Create a FluentdWriter
	fluentWriter := &fluentdWriter{
		conn:   fluentInstance,
		config: c,
	}

	// Set up logrus to use the FluentdWriter
	log.SetFormatter(c.LogFormatter)
	log.SetOutput(fluentWriter)

	uLogger := &UkerLogger{
		config: c,
		writer: fluentWriter,
		Logger: log,
	}

	go uLogger.monitorConnection(c.TestConnectionTime)

	return uLogger
}

func (u *UkerLogger) checkConnection() bool {
	// Enviar un mensaje de prueba a Fluentd para verificar la conexión
	err := u.writer.conn.Post("test", map[string]string{"message": "ping"})
	return err == nil
}

func (u *UkerLogger) stablishFluentDConnection() error {
	fluentInstance, err := fluent.New(u.config.FluentConfig)

	if err != nil {
		return err
	}

	u.writer.conn = fluentInstance
	return nil
}

func (u *UkerLogger) reconnectFluentD() error {
	// Intentar restablecer la conexión
	if u.writer.conn != nil {
		u.writer.conn.Close()
	}
	return u.stablishFluentDConnection()
}

func (u *UkerLogger) monitorConnection(interval time.Duration) {
	for {
		time.Sleep(interval)
		if u.writer.conn == nil || !u.checkConnection() {
			fmt.Println("Connection to FluentD lost. Attempting to reconnect...")
			RetryWithBackoff(u.reconnectFluentD, "Failed to reconnect, retrying in: ", "Reconnected successfully", "seconds")
		}
	}
}
