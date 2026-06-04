package utils

import (
	"strconv"
	"strings"

	v2 "github.com/Netcracker/qubership-cassandra-supplementary/api/v1alpha1"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/constants"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/core"
	coreUtils "github.com/Netcracker/qubership-nosqldb-operator-core/pkg/utils"
	v11 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type BasicLabels struct {
	AppName              string
	AppComponent         string
	AppTechnology        string
	AppPartOf            string
	AppManagedBy         string
	AppManagedByOperator string
}

func (s BasicLabels) GetLabels(ctx core.ExecutionContext) map[string]string {
	spec := ctx.Get(constants.ContextSpec).(*v2.CassandraSupplService)
	labels := map[string]string{
		AppInstance: spec.Spec.Instance,
		AppVersion:  spec.Spec.ArtifactDescriptorVersion,
	}
	extraLabels := map[string]string{
		AppName:              s.AppName,
		AppComponent:         s.AppComponent,
		AppTechnology:        s.AppTechnology,
		AppPartOf:            s.AppPartOf,
		AppManagedBy:         s.AppManagedBy,
		AppManagedByOperator: s.AppManagedByOperator,
	}

	for key, value := range extraLabels {
		if value != "" {
			labels[key] = value
		}
	}
	return labels
}

func CreateRuntimeObjectContextWrapper(
	ctx core.ExecutionContext,
	object client.Object,
	meta v12.ObjectMeta,
	labels BasicLabels,
) error {
	spec := ctx.Get(constants.ContextSpec).(*v2.CassandraSupplService)
	switch obj := any(object).(type) {
	case *v11.Deployment:
		var tolerations []v1.Toleration
		if spec.Spec.Policies != nil {
			tolerations = spec.Spec.Policies.Tolerations
		}
		obj.Spec.Template.Spec.Tolerations = tolerations
		obj.Spec.Template.Spec.SecurityContext = spec.Spec.PodSecurityContext
		obj.Spec.Template.Spec.ServiceAccountName = spec.Spec.ServiceAccountName
		obj.Spec.Template.Spec.PriorityClassName = spec.Spec.Dbaas.PriorityClassName
		for _, container := range obj.Spec.Template.Spec.Containers {
			container.ImagePullPolicy = spec.Spec.ImagePullPolicy
		}
		//fill labels
		for key, value := range labels.GetLabels(ctx) {
			obj.ObjectMeta.Labels[key] = value
			obj.Spec.Template.ObjectMeta.Labels[key] = value
		}
	case *v1.Service:
		for key, value := range labels.GetLabels(ctx) {
			obj.ObjectMeta.Labels[key] = value
		}
	}
	return createRuntimeObjectContextWrapper(ctx, object, meta)
}

// todo last two args can be replaced with one - object
func createRuntimeObjectContextWrapper(ctx core.ExecutionContext, object client.Object, meta v12.ObjectMeta) error {
	scheme := ctx.Get(constants.ContextSchema).(*runtime.Scheme)
	// spec := ctx.Get(constants.ContextSpec).(*v12.DbaasRedisAdapter)
	helper := ctx.Get(constants.KubernetesHelperImpl).(core.KubernetesHelper)
	// specPointer := &(*spec)

	return helper.CreateRuntimeObject(scheme, nil, object, meta)
}

func TLSClientSpecUpdate(depl *v1.PodSpec, rootCertPath string, tls v2.TLS) {
	if !tls.Enabled {
		return
	}
	volProj := []v1.VolumeProjection{
		{
			Secret: &v1.SecretProjection{
				LocalObjectReference: v1.LocalObjectReference{
					Name: tls.RootCASecretName,
				},
				Items: []v1.KeyToPath{
					{
						Path: tls.RootCAFileName,
						Key:  tls.RootCAFileName,
					},
				},
			},
		},
	}

	volume := []v1.Volume{
		{
			Name: RootCert,
			VolumeSource: v1.VolumeSource{
				Projected: &v1.ProjectedVolumeSource{
					Sources:     volProj,
					DefaultMode: nil,
				},
			},
		},
	}

	volumeMount := []v1.VolumeMount{{
		Name:      RootCert,
		MountPath: rootCertPath,
	}}

	depl.Volumes = append(depl.Volumes, volume...)
	depl.Containers[0].VolumeMounts = append(depl.Containers[0].VolumeMounts, volumeMount...)

	depl.Containers[0].Env = append(depl.Containers[0].Env,
		coreUtils.GetPlainTextEnvVar("TLS_ENABLED", strconv.FormatBool(tls.Enabled)),
		coreUtils.GetPlainTextEnvVar("TLS_ROOTCERT", rootCertPath+tls.RootCAFileName),
	)
}

func TLSServerSpecUpdate(depl *v1.PodSpec, tls v2.TLS, secretName, mountPath string) {
	if !tls.Enabled {
		return
	}

	depl.Volumes = append(depl.Volumes,
		v1.Volume{
			Name: secretName,
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: secretName,
				},
			},
		},
	)

	depl.Containers[0].VolumeMounts = append(depl.Containers[0].VolumeMounts,
		v1.VolumeMount{
			Name:      secretName,
			ReadOnly:  true,
			MountPath: mountPath,
		},
	)

	depl.Containers[0].Env = append(depl.Containers[0].Env,
		coreUtils.GetPlainTextEnvVar("INTERNAL_TLS_ENABLED", strconv.FormatBool(tls.Enabled)),
		coreUtils.GetPlainTextEnvVar("INTERNAL_TLS_ROOTCERT", mountPath+tls.RootCAFileName),
		coreUtils.GetPlainTextEnvVar("INTERNAL_TLS_CERTIFICATE_FILENAME", mountPath+tls.SignedCRTFileName),
		coreUtils.GetPlainTextEnvVar("INTERNAL_TLS_KEY_FILENAME", mountPath+tls.PrivateKeyFileName),
		coreUtils.GetPlainTextEnvVar("INTERNAL_TLS_PATH", mountPath),
	)
}

func GetHTTPPort(tlsEnabled bool) int32 {
	var port int32 = 8080
	if tlsEnabled {
		port = 8443
	}
	return port
}

func GetHTTPProtocol(tlsEnabled bool) string {
	if tlsEnabled {
		return "https"
	}
	return "http"
}

func GetUriScheme(tlsEnabled bool) v1.URIScheme {
	if tlsEnabled {
		return v1.URISchemeHTTPS
	}
	return v1.URISchemeHTTP
}

func IsTLSEnableForDBAAS(aggregatorRegistrationAddress string, tlsEnabled bool) bool {
	if !strings.Contains(aggregatorRegistrationAddress, "https") {
		return false
	}

	return tlsEnabled
}

func SanitizeName(name string) string {
	return strings.ReplaceAll(name, ".", "-")
}
