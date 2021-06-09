/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package utils

import (
	// "fmt"
	"bytes"
	"io/ioutil"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	quantity "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8Yaml "k8s.io/apimachinery/pkg/util/yaml"

	v1alpha1 "github.com/jkremser/podinfo-operator/api/v1alpha1"
)

const PodinfoVersion = "5.2.1"

func YamlToDeployment(deploymentManifest []byte) (*appsv1.Deployment, error) {
	d := &appsv1.Deployment{}
	dec := k8Yaml.NewYAMLOrJSONDecoder(bytes.NewReader([]byte(deploymentManifest)), 10000)

	if err := dec.Decode(&d); err != nil {
		return nil, err
	}
	return d, nil
}

func PodinfoDeployment(podinfo *v1alpha1.Podinfo, backend bool) *appsv1.Deployment {
	imgSuffix := "-fe"
	if backend {
		imgSuffix = "-be"
	}

	labels := map[string]string{
		"app": podinfo.Name + imgSuffix,
	}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podinfo.Name + imgSuffix,
			Namespace: podinfo.Namespace,
			Labels:    labels,
		},

		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
					Annotations: map[string]string{
						"prometheus.io/scrape": "true",
						"prometheus.io/port":   "9797",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: "ghcr.io/stefanprodan/podinfo:" + PodinfoVersion,
						Name:  "podinfo",

						Env: []corev1.EnvVar{{
							Name:  "PODINFO_UI_COLOR",
							Value: "#34577c",
						}, {
							Name:  "PODINFO_UI_MESSAGE",
							Value: podinfo.Spec.Message,
						},
						},

						Ports: []corev1.ContainerPort{{
							ContainerPort: 9898,
							Name:          "http",
						}, {
							ContainerPort: 9797,
							Name:          "http-metrics",
						}, {
							ContainerPort: 9999,
							Name:          "grpc",
						},
						},
						Command: []string{
							"./podinfo",
							"--port=9898",
							"--port-metrics=9797",
							"--level=info",
							"--backend-url=http://" + podinfo.Name + "-be:9898/echo",
						},
						LivenessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								Exec: &corev1.ExecAction{
									Command: []string{
										"podcli",
										"check",
										"http",
										"localhost:9898/healthz",
									},
								},
							},
							InitialDelaySeconds: 5,
							TimeoutSeconds:      5,
						},
						ReadinessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								Exec: &corev1.ExecAction{
									Command: []string{
										"podcli",
										"check",
										"http",
										"localhost:9898/readyz",
									},
								},
							},
							InitialDelaySeconds: 5,
							TimeoutSeconds:      5,
						},
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    quantity.MustParse("1000m"),
								corev1.ResourceMemory: quantity.MustParse("128Mi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    quantity.MustParse("100m"),
								corev1.ResourceMemory: quantity.MustParse("32Mi"),
							},
						},
					},
					},
				},
			},
		},
	}

	if backend {
		// override the command for backend deployment
		dep.Spec.Template.Spec.Containers[0].Command = []string{
			"./podinfo",
			"--port=9898",
			"--port-metrics=9797",
			"--level=info",
			"--grpc-port=9999",
			"--grpc-service-name=" + podinfo.Name + "-be",
		}
		// override the Resources.Limits to follow https://github.com/stefanprodan/podinfo/blob/master/deploy/webapp/backend/deployment.yaml
		dep.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceCPU] = quantity.MustParse("2000m")
		dep.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceMemory] = quantity.MustParse("512Mi")
		backend_replicas := int32(podinfo.Spec.BackendReplicas)
		dep.Spec.Replicas = &backend_replicas
	} else {
		frontend_replicas := int32(podinfo.Spec.FrontendReplicas)
		dep.Spec.Replicas = &frontend_replicas
	}

	return dep
}

func PodinfoService(podinfo *v1alpha1.Podinfo, backend bool) *corev1.Service {

	// apiVersion: v1
	// kind: Service
	// metadata:
	//   name: backend
	//   namespace: webapp
	// spec:
	//   type: ClusterIP
	//   selector:
	// 	app: backend
	//   ports:
	// 	- name: http
	// 	  port: 9898
	// 	  protocol: TCP
	// 	  targetPort: http
	// 	- port: 9999
	// 	  targetPort: grpc
	// 	  protocol: TCP
	// 	  name: grpc

	imgSuffix := "-fe"
	if backend {
		imgSuffix = "-be"
	}
	labels := map[string]string{
		"app": podinfo.Name + imgSuffix,
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podinfo.Name + imgSuffix,
			Namespace: podinfo.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: labels,
		},
	}
	if backend {
		svc.Spec.Ports = []corev1.ServicePort{
			{
				Port:       9898,
				Name:       "http",
				TargetPort: intstr.FromString("http"),
			}, {
				Port:       9999,
				Name:       "grpc",
				TargetPort: intstr.FromString("grpc"),
			},
		}
	} else {
		svc.Spec.Ports = []corev1.ServicePort{
			{
				Port:       80,
				Name:       "http",
				Protocol:   corev1.ProtocolTCP,
				TargetPort: intstr.FromString("http"),
			},
		}
	}

	return svc
}

func GetDeployment(name string, namespace string, replicas int32, msg string) (*appsv1.Deployment, error) {
	data, err := ioutil.ReadFile("./resources/deployment.yaml")
	if err != nil {
		panic(err)
	}
	deployment, e := YamlToDeployment(data)
	deployment.Name = name
	deployment.ObjectMeta.Name = name
	deployment.Namespace = namespace
	deployment.Spec.Replicas = &replicas
	deployment.Spec.Selector.MatchLabels["app"] = name
	if msg != "" {
		infoMsgEnv := corev1.EnvVar{
			Name:  "PODINFO_UI_MESSAGE",
			Value: msg,
		}
		deployment.Spec.Template.Spec.Containers[0].Env = append(deployment.Spec.Template.Spec.Containers[0].Env, infoMsgEnv)
	}
	return deployment, e
}

func FrontendDeployment() (*appsv1.Deployment, error) {
	return nil, nil
}
