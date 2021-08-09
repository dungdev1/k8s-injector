package webhook

import (
	"context"
	"net/http"
	"strconv"

	"github.com/dungdev1/k8s-injector/pkg/config"
	"github.com/julienschmidt/httprouter"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var podResource = metav1.GroupVersionResource{Version: "v1", Resource: "pods"}

type WebhookServer struct {
	server          *http.Server
	lifecycleServer *http.Server
	InjConfigs      map[string]*config.InjectionConfig
	Namespaces      map[string]bool
}

func NewWebhookServer() *WebhookServer {
	return &WebhookServer{}
}

func (webhook *WebhookServer) StartInjectorServer(port int, tlsCert string, tlsKey string) error {
	webhook.server = &http.Server{
		Addr:    ":" + strconv.Itoa(port),
		Handler: webhook.bootRouter(),
	}

	return webhook.server.ListenAndServeTLS(tlsCert, tlsKey)
}

func (webhook *WebhookServer) StartLifeCycleServer(port int) error {
	webhook.lifecycleServer = &http.Server{
		Addr:    ":" + strconv.Itoa(port),
		Handler: webhook.lifeCycleBootRouter(),
	}

	return webhook.server.ListenAndServe()
}

func (webhook *WebhookServer) Shutdown() error {
	return webhook.server.Shutdown(context.Background())
}

func (webhook *WebhookServer) bootRouter() *httprouter.Router {
	router := httprouter.New()

	router.POST("/mutate", webhook.Mutate)

	return router
}

func (webhook *WebhookServer) lifeCycleBootRouter() *httprouter.Router {
	router := httprouter.New()

	router.GET("/healthz", webhook.Health)

	return router
}
