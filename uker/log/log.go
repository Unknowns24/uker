package log

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/fluent/fluent-logger-golang/fluent"
	"github.com/sirupsen/logrus"
)

// FluentMetadata groups common metadata fields sent to Fluentd.
type FluentMetadata struct {
	Tag         string
	Source      string
	ServiceName string
	Application string
}

// Config configures the logger helper.
type Config struct {
	FluentMetadata     FluentMetadata
	FluentConfig       fluent.Config
	LogFormatter       logrus.Formatter
	LogOnConsole       bool
	TestConnectionTime time.Duration
}

// Logger wraps a logrus logger that writes into Fluentd.
type Logger struct {
	config *Config
	writer *fluentWriter
	Logger *logrus.Logger
}

type fluentWriter struct {
	conn   *fluent.Fluent
	config *Config
}

func (fw *fluentWriter) Write(p []byte) (int, error) {
	payload := map[string]any{}
	if err := json.Unmarshal(p, &payload); err != nil {
		return 0, err
	}

	metadata := map[string]string{
		"application": fw.config.FluentMetadata.Application,
		"servicename": fw.config.FluentMetadata.ServiceName,
		"source":      fw.config.FluentMetadata.Source,
	}

	for key, value := range metadata {
		if value != "" {
			payload[key] = value
		}
	}

	if fw.config.LogOnConsole {
		fmt.Println(payload["msg"])
	}

	if err := fw.conn.Post(fw.config.FluentMetadata.Tag, payload); err != nil {
		return 0, err
	}

	return len(p), nil
}

// New creates and configures a logger that writes to Fluentd.
func New(config *Config) *Logger {
	fluentInstance, err := fluent.New(config.FluentConfig)
	if err != nil {
		panic(fmt.Sprintf("failed to connect to Fluentd: %v", err))
	}

	writer := &fluentWriter{conn: fluentInstance, config: config}
	log := logrus.New()
	log.SetFormatter(config.LogFormatter)
	log.SetOutput(writer)

	logger := &Logger{config: config, writer: writer, Logger: log}
	go logger.monitorConnection(config.TestConnectionTime)
	return logger
}

func (l *Logger) monitorConnection(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		if l.writer.conn == nil || !l.checkConnection() {
			fmt.Println("Connection to FluentD lost. Attempting to reconnect...")
			RetryWithBackoff(func() error {
				if err := l.reconnectFluentD(); err != nil {
					fmt.Printf("Failed to reconnect: %v\n", err)
					return err
				}
				fmt.Println("Reconnected successfully")
				return nil
			})
		}
	}
}

func (l *Logger) checkConnection() bool {
	return l.writer.conn.Post("test", map[string]string{"message": "ping"}) == nil
}

func (l *Logger) reconnectFluentD() error {
	if l.writer.conn != nil {
		_ = l.writer.conn.Close()
	}

	conn, err := fluent.New(l.config.FluentConfig)
	if err != nil {
		return err
	}

	l.writer.conn = conn
	return nil
}
