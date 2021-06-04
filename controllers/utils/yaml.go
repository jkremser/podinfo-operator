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
	k8Yaml "k8s.io/apimachinery/pkg/util/yaml"
)

func YamlToDeployment(deploymentManifest []byte) (*appsv1.Deployment, error) {
	d := &appsv1.Deployment{}
	dec := k8Yaml.NewYAMLOrJSONDecoder(bytes.NewReader([]byte(deploymentManifest)), 1000)

	if err := dec.Decode(&d); err != nil {
		return nil, err
	}
	return d, nil
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
