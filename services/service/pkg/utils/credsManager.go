package utils

import (
	"github.com/Netcracker/qubership-credential-manager/pkg/manager"
	v1 "k8s.io/api/core/v1"
)

type CredsManagerI interface {
	AddCredHashToPodTemplate(secretNames []string, template *v1.PodTemplateSpec) error
}

type CredsManager struct {
}

func (c *CredsManager) AddCredHashToPodTemplate(secretNames []string, template *v1.PodTemplateSpec) error {
	return manager.AddCredHashToPodTemplate(secretNames, template)
}
