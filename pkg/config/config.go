package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"k8s.io/client-go/util/homedir"
)

const (
	lifeCyclePortConfigKey       = "LIFE_CYCLE_PORT"
	lifecyclePortDefault         = 8000
	tlsPortConfigKey             = "TLS_PORT"
	tlsPortDefault               = 9443
	tlsCertFileConfigKey         = "TLS_CERTIFICATE_FILE"
	tlsCertFileDefault           = "/var/lib/secrets/cert.pem"
	tlsKeyFileConfigKey          = "TLS_KEY_FILE"
	tlsKeyFileDefault            = "/var/lib/secrets/key.pem"
	annotationNamespaceConfigKey = "ANNOTATION_NAMESPACE"
	annotationNamespaceDefault   = ""
	configmapNameConfigKey       = "CONFIGMAP_NAME"
	configmapNamespaceConfigKey  = "CONFIGMAP_NAMESPACE"
	configmapNamespaceDefault    = ""
	logLevelConfigKey            = "LOG_LEVEL"
	logLevelConfigDefault        = "info"
	kubeConfigConfigKey          = "KUBE_CONFIG"
	masterUrlConfigKey           = "MASTER_URL"
)

type Config struct {
	LifecyclePort       int
	TLSPort             int
	CertFile            string
	KeyFile             string
	AnnotationNamespace string
	ConfigmapNamespace  string
	ConfigMapName       string
	LogLevel            string
	KubeConfig          string
	MasterURL           string
	WebhookEnableLabel  map[string]string
}

const (
	ServiceAccountNamespaceFilePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
)

func ParseCliArgs(config *Config) error {
	webhookEnableLabel := NewMapStringStringFlag()

	flag.IntVar(&config.LifecyclePort, "lifecycle-port", getIntEnv(lifeCyclePortConfigKey, lifecyclePortDefault), "Port for health checking (http only)")
	flag.IntVar(&config.TLSPort, "tls-port", getIntEnv(tlsPortConfigKey, tlsPortDefault), "Webhook server port for handling admission controller request (forced https)")
	flag.StringVar(&config.CertFile, "tls-cert-file", getEnv(tlsCertFileConfigKey, tlsCertFileDefault), "File containing the x509 certificate of server")
	flag.StringVar(&config.KeyFile, "tls-key-file", getEnv(tlsKeyFileConfigKey, tlsKeyFileDefault), "File containing the x509 private key of server")
	flag.StringVar(&config.AnnotationNamespace, "annotation-namespace", getEnv(annotationNamespaceConfigKey, annotationNamespaceDefault), "The annotation namespace")
	flag.StringVar(&config.ConfigmapNamespace, "configmap-namespace", getEnv(configmapNamespaceConfigKey, configmapNamespaceDefault), "Namespace to search for ConfigMap to load Injection Config from (default: current namespace")
	flag.StringVar(&config.ConfigMapName, "configmap-name", getEnv(configmapNameConfigKey, ""), "Name of ConfigMap to load Injection Config from")
	flag.StringVar(&config.LogLevel, "log-level", getEnv(logLevelConfigKey, logLevelConfigDefault), "Sets the log level (DEBUG, INFO, ERROR, ...)")
	flag.StringVar(&config.KubeConfig, "kube-config", getEnv(kubeConfigConfigKey, ""), "Path contain the config for kubernetes cluster")
	flag.StringVar(&config.MasterURL, "master-url", getEnv(masterUrlConfigKey, ""), "master url of kubernetes cluster")
	flag.Var(&webhookEnableLabel, "webhook-enable-label", "Label pair used to enable this webhook on namespace")
	flag.Parse()

	config.WebhookEnableLabel = webhookEnableLabel.ToMapStringString()
	if len(config.WebhookEnableLabel) == 0 {
		config.WebhookEnableLabel["k8s-injection"] = "enabled"
	}

	switch strings.ToLower(config.LogLevel) {
	case "info":
	case "debug":
	case "error":
	default:
		return fmt.Errorf("invalid log-level passed: %s Should be one of: info, debug, error", config.LogLevel)
	}

	if config.ConfigmapNamespace == "" {
		ns, err := os.ReadFile(ServiceAccountNamespaceFilePath)
		if err != nil {
			config.ConfigmapNamespace = "default"
		}
		if string(ns) != "" {
			config.ConfigmapNamespace = string(ns)
		}
	}

	if config.ConfigMapName == "" {
		return fmt.Errorf("configmap name not found, this argument is mandatory")
	}

	if home := homedir.HomeDir(); config.KubeConfig == "" && home != "" {
		config.KubeConfig = filepath.Join(home, ".kube", "config")
	}

	config.PrintHumanConfigArgs()

	return nil
}

func (c *Config) PrintHumanConfigArgs() {
	log.Info().Msgf(
		"k8s-injector arguments: \n"+
			"\tlifecycle-port: %d,\n"+
			"\ttls-port: %d,\n"+
			"\ttls-cert-file: %s\n"+
			"\ttls-key-file: %s\n"+
			"\tannotation-namespace: %s\n"+
			"\tconfigmap-name: %s\n"+
			"\tconfigmap-namespace: %s\n"+
			"\tlog-level: %s\n"+
			"\tkube-config: %s\n"+
			"\tmaster-url: %s\n"+
			"\twebhook-enable-label: %s\n",
		c.LifecyclePort,
		c.TLSPort,
		c.CertFile,
		c.KeyFile,
		c.AnnotationNamespace,
		c.ConfigMapName,
		c.ConfigmapNamespace,
		c.LogLevel,
		c.KubeConfig,
		c.MasterURL,
		c.WebhookEnableLabel,
	)
}

func getEnv(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		if value != "" {
			return value
		}
	}
	return fallback
}
func getIntEnv(key string, fallback int) int {
	envStrValue := getEnv(key, "")
	if envStrValue == "" {
		return fallback
	}
	envIntValue, err := strconv.Atoi(envStrValue)
	if err != nil {
		panic("Env Var " + key + " must be an integer")
	}
	return envIntValue
}
