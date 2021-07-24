package watcher

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/dungdev1/k8s-injector/pkg/config"
	"github.com/rs/zerolog/log"
	v1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	k8sv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type K8sWatcher struct {
	Namespace  string
	CfmName    string
	MasterURL  string
	KubeConfig string
	client     k8sv1.CoreV1Interface
}

type NamespaceEvent struct {
	Namespace string
	Type      watch.EventType
}

var ErrWatcheChannelClosed = errors.New("watcher channel has close")

func NewK8sWatcher(ns string, cfmName string, masterUrl string, kubeconfig string) (*K8sWatcher, error) {
	c := K8sWatcher{
		CfmName:    cfmName,
		Namespace:  ns,
		MasterURL:  masterUrl,
		KubeConfig: kubeconfig,
	}

	var k8sConfig *rest.Config
	var err error
	if masterUrl != "" {
		log.Info().Msgf("Use master url: %s for connecting to cluster.", masterUrl)
		log.Info().Msgf("Creating Kubernetes client from kubeconfig=%s with masterUrl=%s", c.KubeConfig, c.MasterURL)
		k8sConfig, err = clientcmd.BuildConfigFromFlags(c.MasterURL, c.KubeConfig)
		if err != nil {
			return nil, fmt.Errorf("cannot create Kubernetes client from outside cluster with error: %s", err)
		}
	} else {
		if c.Namespace == "" {
			ns, err := ioutil.ReadFile(config.ServiceAccountNamespaceFilePath)
			if err != nil {
				c.Namespace = "default"
			}
			if string(ns) != "" {
				c.Namespace = string(ns)
				log.Info().Msgf("Use current namespace=%s from %s for searching configmap", c.Namespace, config.ServiceAccountNamespaceFilePath)
			}
		}
		log.Info().Msg("Creating Kubernetes client from in-cluster discovery")
		k8sConfig, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("cannot create Kubernetes client from in-cluster with error %s", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, err
	}
	c.client = clientset.CoreV1()
	log.Info().Msgf("Created watcher: apiserver=%s, namespace=%s", k8sConfig.Host, c.Namespace)
	return &c, nil
}

func (c *K8sWatcher) WatchNamespace(ctx context.Context, webhookEnabledLabel map[string]string, ch chan<- NamespaceEvent) error {
	log.Info().Msg("Watching for all namespace in cluster...")

	labelSelector := metav1.LabelSelector{MatchLabels: webhookEnabledLabel}
	watcher, err := c.client.Namespaces().Watch(ctx, metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	})
	if err != nil {
		return fmt.Errorf("failed to start Namespace watcher: %s", err.Error())
	}

	for {
		select {
		case e, ok := <-watcher.ResultChan():
			if !ok {
				return ErrWatcheChannelClosed
			}
			if e.Type == watch.Error {
				return apierrs.FromObject(e.Object)
			}
			namespace, ok := e.Object.(*v1.Namespace)
			if !ok {
				log.Error().Msg("cannot parse the event")
				break
			}
			switch e.Type {
			case watch.Added:
				log.Info().Msgf("Added a Namespace: %s", namespace.Name)
				ch <- NamespaceEvent{
					Namespace: namespace.Name,
					Type:      watch.Added,
				}
			case watch.Deleted:
				log.Info().Msgf("Deleted label or removed namespace %s", namespace.Name)
				ch <- NamespaceEvent{
					Namespace: namespace.Name,
					Type:      watch.Deleted,
				}
			}
		case <-ctx.Done():
			log.Info().Msg("stopping namespace watcher, context indicated we are done")
			return nil
		}
	}
}

func (c *K8sWatcher) WatchConfigMap(ctx context.Context, notify chan<- interface{}) error {
	log.Info().Msgf("Watching for configmap=%s on namespace=%s", c.CfmName, c.Namespace)
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{"app": "k8s-injector"}}
	watcher, err := c.client.ConfigMaps(c.Namespace).Watch(ctx, metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	})
	if err != nil {
		return fmt.Errorf("failed to start ConfigMap watcher: %s", err.Error())
	}
	for {
		select {
		case e, ok := <-watcher.ResultChan():
			if !ok {
				return ErrWatcheChannelClosed
			}
			if e.Type == watch.Error {
				return apierrs.FromObject(e.Object)
			}
			configmap, ok := e.Object.(*v1.ConfigMap)
			if !ok {
				log.Error().Msg("cannot parse the event")
				break
			}
			if configmap.Name != c.CfmName {
				break
			}
			switch e.Type {
			case watch.Added:
				fallthrough
			case watch.Modified:
				fallthrough
			case watch.Deleted:
				notify <- struct{}{}
			default:
				log.Error().Msgf("go unsupported event %s for %s! skipping", e.Type, e.Object.GetObjectKind())
			}
		case <-ctx.Done():
			log.Info().Msg("stopping configmap watcher, context indicated we are done")
			return nil
		}
	}
}

func (c *K8sWatcher) GetConfigMap(ctx context.Context) (map[string]*config.InjectionConfig, error) {
	log.Info().Msg("Fetching Configmaps...")
	cfm, err := c.client.ConfigMaps(c.Namespace).Get(ctx, c.CfmName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("cannot get config map with error: %s", err.Error())
	}
	injs := map[string]*config.InjectionConfig{}
	for cfmFile, payload := range cfm.Data {
		inj, err := config.LoadInjectionConfig([]byte(payload))
		if err != nil {
			log.Error().Msgf("cannot load injection config from ConfigMap: %s with error: %s", cfmFile, err.Error())
			continue
		}
		path := strings.ReplaceAll(cfmFile, ".", "/")
		injs[path] = inj
	}
	return injs, nil
}
