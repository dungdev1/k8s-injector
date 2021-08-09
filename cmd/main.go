package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/dungdev1/k8s-injector/pkg/config"
	webhook "github.com/dungdev1/k8s-injector/pkg/server"
	watcherpkg "github.com/dungdev1/k8s-injector/pkg/watcher"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/watch"
)

const timeFormat = "02/01/2006 15:04:05"

var mainConfig config.Config

func init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: timeFormat, NoColor: true})

	err := config.ParseCliArgs(&mainConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to parse cli args")
	}

	switch strings.ToLower(mainConfig.LogLevel) {
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	}
}

var InjConfigs = map[string]*config.InjectionConfig{}

func main() {

	// Start web server
	webhook := webhook.NewWebhookServer()

	// Start up the watcher, and get configMaps
	watcher, err := watcherpkg.NewK8sWatcher(mainConfig.ConfigmapNamespace, mainConfig.ConfigMapName, mainConfig.MasterURL, mainConfig.KubeConfig)
	if err != nil {
		panic(err.Error())
	}
	ctx := context.Background()

	namespaces := make(map[string]bool)

	namespaceEventChan := make(chan watcherpkg.NamespaceEvent)
	go func() {
		for {
			err = watcher.WatchNamespace(ctx, mainConfig.WebhookEnableLabel, namespaceEventChan)
			if err != nil {
				switch err {
				case watcherpkg.ErrWatcheChannelClosed:
					log.Info().Msgf("Namespace watcher got error: %s, Restart Namespace watcher", err.Error())
				default:
					panic(err.Error())
				}
			}
		}
	}()

	cfmEventChan := make(chan interface{})
	go func() {
		for {
			err = watcher.WatchConfigMap(ctx, cfmEventChan)
			if err != nil {
				switch err {
				case watcherpkg.ErrWatcheChannelClosed:
					log.Info().Msgf("ConfigMap watcher got error: %s, Restart ConfigMap watcher", err.Error())
				default:
					panic(err.Error())
				}
			}
		}
	}()

	go func() {
		for range time.NewTicker(1 * time.Second).C {
			select {
			case nsEvent := <-namespaceEventChan:
				log.Info().Msg("Received namespace event")
				if nsEvent.Type == watch.Added {
					if namespaces[nsEvent.Namespace] {
						break
					}
					namespaces[nsEvent.Namespace] = true
					log.Info().Msgf("Added namespace %q to namespace list: %v", nsEvent.Namespace, namespaces)
				} else if nsEvent.Type == watch.Deleted {
					delete(namespaces, nsEvent.Namespace)
					log.Info().Msgf("Removed namespace %q from namespace list: %v", nsEvent.Namespace, namespaces)
				}
				webhook.Namespaces = namespaces
			case <-cfmEventChan:
				log.Info().Msg("Received configmap event")
				injConfigs, err := watcher.GetConfigMap(ctx)
				if err != nil {
					panic(err.Error())
				}
				log.Info().Msgf("Fetched configmap %q in namespace %q", watcher.CfmName, watcher.Namespace)
				webhook.InjConfigs = injConfigs
			}
		}
	}()

	// listening OS shutdown signal
	go func() {
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, syscall.SIGTERM)
		<-signalChan
		log.Info().Msg("Received SIGTERM, shuting down...")
		if err := webhook.Shutdown(); err != nil {
			log.Fatal().Msgf("Failed to shutdown server: %v", err.Error())
		}
		os.Exit(0)
	}()

	log.Info().Msgf("Service is ready to listen on port: %d", mainConfig.TLSPort)
	if err := webhook.StartInjectorServer(mainConfig.TLSPort, mainConfig.CertFile, mainConfig.KeyFile); err != nil {
		log.Fatal().Msgf("Service failed: %v", err.Error())
	}
	log.Info().Msgf("Started webhook server on port %s", mainConfig.TLSPort)

	if err := webhook.StartLifeCycleServer((mainConfig.LifecyclePort)); err != nil {
		log.Fatal().Msgf("Service failed: %v", err.Error())
	}
	log.Info().Msgf("Started lifecycle server on port %s", mainConfig.LifecyclePort)
}
