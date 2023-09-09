package webhook

import (
	"net/http"

	"github.com/dungdev1/k8s-injector/pkg/controller"
	"github.com/julienschmidt/httprouter"
	"github.com/rs/zerolog/log"
)

func (webhook *WebhookServer) Mutate(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	log.Info().Msg("Handling mutating request...")

	var writeErr error

	if bytes, err := controller.AdmissionControllerHandler(w, r, controller.ApplyNewConfig, controller.MutatingAdmission, webhook.InjConfigs, webhook.Namespaces); err != nil {
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

func (webhook *WebhookServer) Validate(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	log.Info().Msg("Handling validating request...")
}

func (webhook *WebhookServer) Health(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	log.Debug().Msg("Handling health checking request...")

	w.WriteHeader(http.StatusOK)
	_, writeErr := w.Write([]byte("ok"))
	if writeErr != nil {
		log.Info().Msgf("Could not write response: %v", writeErr)
	}
}
