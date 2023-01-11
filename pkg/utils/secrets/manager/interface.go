// Copyright (c) 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package manager

import (
	"context"

	secretsutils "github.com/gardener/gardener/pkg/utils/secrets"

	corev1 "k8s.io/api/core/v1"
)

// Reader is part of the SecretsManager interface and allows retrieving secrets from a SecretsManager.
type Reader interface {
	// Get returns the secret object for the secret with the given name. By default, the bundle secret will be returned.
	// If there is no bundle secret then it falls back to the current secret. Note that only those secrets are known
	// which were detected or generated by prior Generate calls.
	Get(string, ...GetOption) (*corev1.Secret, bool)
}

// Interface describes the methods for managing secrets.
type Interface interface {
	// Generate generates a secret based on the provided configuration. If the secret for the provided configuration
	// already exists then it is returned with re-generation. The function also automatically rotates/re-generates the
	// secret only if necessary (e.g., when the config or the signing CA changes).
	Generate(context.Context, secretsutils.ConfigInterface, ...GenerateOption) (*corev1.Secret, error)

	Reader

	// Cleanup deletes no longer required secrets. No longer required secrets are those still existing in the system
	// which weren't detected by prior Generate calls. Consequently, only call Cleanup after you have executed Generate
	// calls for all desired secrets.
	Cleanup(context.Context) error
}
