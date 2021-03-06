/*
Copyright 2020 kubeflow.org.

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

package v1beta1

import (
	"fmt"
	"strconv"

	"github.com/kubeflow/kfserving/pkg/constants"
	"github.com/kubeflow/kfserving/pkg/utils"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SKLearnSpec defines arguments for configuring SKLearn model serving.
type SKLearnSpec struct {
	// Contains fields shared across all predictors
	PredictorExtensionSpec `json:",inline"`
}

var _ ComponentImplementation = &SKLearnSpec{}

// Validate returns an error if invalid
func (k *SKLearnSpec) Validate() error {
	return utils.FirstNonNilError([]error{
		validateStorageURI(k.GetStorageUri()),
	})
}

// Default sets defaults on the resource
func (k *SKLearnSpec) Default(config *InferenceServicesConfig) {
	k.Container.Name = constants.InferenceServiceContainerName

	if k.RuntimeVersion == nil {
		defaultVersion := config.Predictors.SKlearn.V1.DefaultImageVersion
		if k.ProtocolVersion != nil && *k.ProtocolVersion == constants.ProtocolV2 {
			defaultVersion = config.Predictors.SKlearn.V2.DefaultImageVersion
		}

		k.RuntimeVersion = &defaultVersion
	}

	if k.ProtocolVersion == nil {
		defaultProtocol := constants.ProtocolV1
		k.ProtocolVersion = &defaultProtocol
	}

	setResourceRequirementDefaults(&k.Resources)
}

// GetContainer transforms the resource into a container spec
func (k *SKLearnSpec) GetContainer(metadata metav1.ObjectMeta, extensions *ComponentExtensionSpec, config *InferenceServicesConfig) *v1.Container {
	if k.ProtocolVersion == nil || *k.ProtocolVersion == constants.ProtocolV1 {
		return k.getContainerV1(metadata, extensions, config)
	}

	return k.getContainerV2(metadata, extensions, config)
}

func (k *SKLearnSpec) getContainerV1(metadata metav1.ObjectMeta, extensions *ComponentExtensionSpec, config *InferenceServicesConfig) *v1.Container {
	arguments := []string{
		fmt.Sprintf("%s=%s", constants.ArgumentModelName, metadata.Name),
		fmt.Sprintf("%s=%s", constants.ArgumentModelDir, constants.DefaultModelLocalMountPath),
		fmt.Sprintf("%s=%s", constants.ArgumentHttpPort, constants.InferenceServiceDefaultHttpPort),
	}

	if extensions.ContainerConcurrency != nil {
		arguments = append(arguments, fmt.Sprintf("%s=%s", constants.ArgumentWorkers, strconv.FormatInt(*extensions.ContainerConcurrency, 10)))
	}

	if k.Container.Image == "" {
		k.Container.Image = config.Predictors.SKlearn.V1.ContainerImage + ":" + *k.RuntimeVersion
	}

	k.Container.Name = constants.InferenceServiceContainerName
	k.Container.Args = arguments
	return &k.Container
}

func (k *SKLearnSpec) getContainerV2(metadata metav1.ObjectMeta, extensions *ComponentExtensionSpec, config *InferenceServicesConfig) *v1.Container {
	k.Container.Env = append(
		k.Container.Env,
		v1.EnvVar{
			Name:  constants.MLServerHTTPPortEnv,
			Value: strconv.Itoa(int(constants.MLServerISRestPort)),
		},
		v1.EnvVar{
			Name:  constants.MLServerGRPCPortEnv,
			Value: strconv.Itoa(int(constants.MLServerISGRPCPort)),
		},
		v1.EnvVar{
			Name:  constants.MLServerModelsDirEnv,
			Value: constants.DefaultModelLocalMountPath,
		},
	)

	// Append fallbacks for model settings
	k.Container.Env = append(
		k.Container.Env,
		k.getDefaultsV2(metadata)...,
	)

	if k.Container.Image == "" {
		k.Container.Image = config.Predictors.SKlearn.V2.ContainerImage + ":" + *k.RuntimeVersion
	}

	return &k.Container
}

func (k *SKLearnSpec) getDefaultsV2(metadata metav1.ObjectMeta) []v1.EnvVar {
	// These env vars set default parameters that can always be overriden
	// individually through `model-settings.json` config files.
	// These will be used as fallbacks for any missing properties and / or to run
	// without a `model-settings.json` file in place.
	return []v1.EnvVar{
		v1.EnvVar{
			Name:  constants.MLServerModelImplementationEnv,
			Value: constants.MLServerSKLearnImplementation,
		},
		v1.EnvVar{
			Name:  constants.MLServerModelNameEnv,
			Value: metadata.Name,
		},
		v1.EnvVar{
			Name:  constants.MLServerModelVersionEnv,
			Value: constants.MLServerModelVersionDefault,
		},
		v1.EnvVar{
			Name:  constants.MLServerModelURIEnv,
			Value: constants.DefaultModelLocalMountPath,
		},
	}
}

func (k *SKLearnSpec) GetStorageUri() *string {
	return k.StorageURI
}

func (k *SKLearnSpec) GetProtocol() constants.InferenceServiceProtocol {
	if k.ProtocolVersion != nil {
		return *k.ProtocolVersion
	} else {
		return constants.ProtocolV1
	}
}
