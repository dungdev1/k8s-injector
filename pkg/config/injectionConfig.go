package config

import (
	"github.com/ghodss/yaml"
	corev1 "k8s.io/api/core/v1"
)

type InjectionConfig struct {
	Name           *string                      `json:"name"`
	Containers     []corev1.Container           `json:"containers"`
	Volumes        []corev1.Volume              `json:"volumes"`
	Environments   []corev1.EnvVar              `json:"env"`
	VolumeMounts   []corev1.VolumeMount         `json:"volumeMounts"`
	HostNetwork    *bool                        `json:"hostNetwork"`
	HostPID        *bool                        `json:"hostPID"`
	InitContainers []corev1.Container           `json:"initContainers"`
	Readiness      *corev1.Probe                `json:"readinessProbe"`
	Liveness       *corev1.Probe                `json:"livenessProbe"`
	Startup        *corev1.Probe                `json:"startupProbe"`
	Resources      *corev1.ResourceRequirements `json:"resources"`
	Ports          []corev1.ContainerPort       `json:"ports"`
	// version        string
}

// func (c *InjectionConfig) String() string {
// 	return fmt.Sprintf("%s: %d containers, %d init containers, %d volumes, %d environment vars, %d volume mounts",
// 		c.FullName(),
// 		len(c.Containers),
// 		len(c.InitContainers),
// 		len(c.Volumes),
// 		len(c.Environments),
// 		len(c.VolumeMounts))
// }

// func (c *InjectionConfig) FullName() string {
// 	return fmt.Sprintf("%s:%s", c.Name, c.Version())
// }

// func (c *InjectionConfig) Version() string {
// 	if c.version == "" {
// 		return defaultVersion
// 	}

// 	return c.version
// }

func LoadInjectionConfig(payload []byte) (*InjectionConfig, error) {
	cfg := InjectionConfig{}
	if err := yaml.Unmarshal(payload, &cfg); err != nil {
		return nil, err
	}
	// if cfg.Name == "" {
	// 	return nil, fmt.Errorf(`name field is required for an injection config`)
	// }

	return &cfg, nil
}
