package main

import (
	"os"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
)

type EnvVars struct {
	ENV       Environment
	PORT      string
	LOG_LEVEL string
}

type Application struct {
	Vars   EnvVars
	logger *log.Logger
	// PORT      string
	// LOG_LEVEL string
}

type Environment string

const (
	EnvLocal Environment = "local"
	EnvDev   Environment = "dev"
	EnvProd  Environment = "prd"
)

func initApp() Application {
	err := godotenv.Overload()

	log_level := getEnv("LOG_LEVEL", "INFO")

	logger := initLogger(log_level)
	if err != nil {
		logger.Warnf("Failed to load .env file: %v", err)
	}

	env := Environment(getEnv("ENV", "local"))
	envVars := EnvVars{
		ENV:  env,
		PORT: getEnv("PORT", "8080"),
	}

	return Application{
		Vars:   envVars,
		logger: logger,
	}
}

func initLogger(logLevel string) *log.Logger {
	logger := log.New()
	logger.SetFormatter(&nested.Formatter{
		HideKeys:        true,
		ShowFullLevel:   true,
		NoColors:        false,
		FieldsOrder:     []string{"module", "function"},
		TimestampFormat: "2006-01-02T15:04:05",
	})
	logger.SetOutput(os.Stdout)
	logger.SetLevel(getLogLevel(logLevel))
	logger.SetReportCaller(true)

	return logger
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultValue
}

func getLogLevel(level string) log.Level {
	switch level {
	case "DEBUG":
		return log.DebugLevel
	case "INFO":
		return log.InfoLevel
	case "WARN":
		return log.WarnLevel
	case "ERROR":
		return log.ErrorLevel
	case "FATAL":
		return log.FatalLevel
	case "PANIC":
		return log.PanicLevel
	default:
		return log.InfoLevel
	}
}
