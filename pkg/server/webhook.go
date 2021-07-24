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
	server     *http.Server
	InjConfigs map[string]*config.InjectionConfig
	Namespaces map[string]bool
}

func NewWebhookServer() *WebhookServer {
	return &WebhookServer{}
}

func (webhook *WebhookServer) Start(port int, tlsCert string, tlsKey string) error {
	webhook.server = &http.Server{
		Addr:    ":" + strconv.Itoa(port),
		Handler: webhook.bootRouter(),
	}

	return webhook.server.ListenAndServeTLS(tlsCert, tlsKey)
	// return webhook.server.ListenAndServe()
}

func (webhook *WebhookServer) Shutdown() error {
	return webhook.server.Shutdown(context.Background())
}

func (webhook *WebhookServer) bootRouter() *httprouter.Router {
	router := httprouter.New()

	router.POST("/mutate", webhook.Mutate)

	return router
}
