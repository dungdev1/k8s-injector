package watcher

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	fakeclient "k8s.io/client-go/kubernetes/fake"
)

func TestWatcher_WatchNamespaceWithAddEvent(t *testing.T) {
	client := fakeclient.NewSimpleClientset()

	w := K8sWatcher{
		Namespace: "kube-system",
		CfmName:   "",
		client:    client.CoreV1(),
	}
	labels := map[string]string{
		"k8s-injection": "enabled",
	}

	ch := make(chan NamespaceEvent)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(1 * time.Second)
		w.client.Namespaces().Create(ctx, &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "dbservice",
				Labels: labels,
			},
		}, metav1.CreateOptions{})
		log.Info().Msg("Namespace \"dbservice\" is created")
	}()

	go func() {
		err := w.WatchNamespace(ctx, labels, ch)
		if err != nil {
			t.Errorf("Watch Namespace err")
		}
	}()

	defer close(ch)

	select {
	case event := <-ch:
		if event.Type != watch.Added {
			t.Errorf("WatchNamespaceWithAddEvent got = %q; want = %q", event.Type, watch.Added)
			break
		}
		log.Info().Msgf("Received namespace %v event with type %v", event.Namespace, event.Type)
	case <-time.After(3 * time.Second):
		t.Error("Fail WatchNamespace test")
	}

	cancel()
}

func TestWatcher_WatchNamespaceWithDeleteEvent(t *testing.T) {
	client := fakeclient.NewSimpleClientset()
	namespace := "dbservice"

	w := K8sWatcher{
		Namespace: "kube-system",
		CfmName:   "k8s-injector",
		client:    client.CoreV1(),
	}

	ch := make(chan NamespaceEvent)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(1 * time.Second)
		// Create namespace
		w.client.Namespaces().Create(ctx, &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
				Labels: map[string]string{
					"k8s-injection": "enabled",
				},
			},
		}, metav1.CreateOptions{})
		log.Info().Msgf("Namespace %s is created", namespace)

		// Delete namespace
		w.client.Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{})
		log.Info().Msgf("Namespace %s is deleted", namespace)
	}()

	go func() {
		err := w.WatchNamespace(ctx, map[string]string{"k8s-injection": "enabled"}, ch)
		if err != nil {
			t.Errorf("Watch Namespace err")
		}
	}()

	defer close(ch)
	defer cancel()
	for {
		select {
		case event := <-ch:
			if event.Type == watch.Deleted {
				log.Info().Msgf("Received namespace %v event with type %v", event.Namespace, event.Type)
				return
			}
		case <-time.After(3 * time.Second):
			t.FailNow()
		}
	}
}

func TestWatcher_GetConfigMap(t *testing.T) {
	// Create configmap
	client := fakeclient.NewSimpleClientset()

	w := K8sWatcher{
		Namespace: "kube-system",
		CfmName:   "k8s-injector",
		client:    client.CoreV1(),
	}

	ctx := context.Background()
	w.client.ConfigMaps(w.Namespace).Create(ctx, &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: w.CfmName,
		},
		Data: map[string]string{
			"abdd--dds": `containers:
- name: healthcheck
  image: 968914998835.dkr.ecr.ap-southeast-1.amazonaws.com/devops:healthcheck-server-13
  ports:
  - containerPort: 3990
  command:
    - /usr/bin/healthcheck-server
  livenessProbe:
    httpGet:
      path: /healthcheck-server/healthz
      port: 3990
    initialDelaySeconds: 5
    timeoutSeconds: 5
    periodSeconds: 10
    successThreshold: 1
    failureThreshold: 3
  imagePullPolicy: IfNotPresent`,
		},
	}, metav1.CreateOptions{})

	_, err := w.GetConfigMap(ctx)
	if err != nil {
		t.Error(err)
	}
}
