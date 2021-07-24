package webhook

import (
	"net/http"

	"github.com/dungdev1/k8s-injector/pkg/admit"
	"github.com/dungdev1/k8s-injector/pkg/controller"
	"github.com/julienschmidt/httprouter"
	"github.com/rs/zerolog/log"
)

func (webhook *WebhookServer) Mutate(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	log.Info().Msg("Handling mutating request...")

	var writeErr error

	if bytes, err := controller.HandleAdmitFunc(w, r, admit.ApplyNewConfig, webhook.InjConfigs, webhook.Namespaces); err != nil {
		log.Error().Msgf("Error handling mutating request: %v", err)
		_, writeErr = w.Write([]byte(err.Error()))
	} else {
		log.Info().Msg("Mutating request handled successfully")
		_, writeErr = w.Write(bytes)
	}

	if writeErr != nil {
		log.Info().Msgf("Could not write response: %v", writeErr)
	}
}
