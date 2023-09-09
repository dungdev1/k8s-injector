package config

import (
	"testing"
)

func TestLoadInjectionConfig(t *testing.T) {
	configPayload := `containers:
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
  imagePullPolicy: IfNotPresent`
	_, err := LoadInjectionConfig([]byte(configPayload))
	if err != nil {
		t.Error(err)
	}
}
