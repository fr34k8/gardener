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

package shoot_test

import (
	"context"
	"errors"

	"github.com/gardener/gardener/extensions/pkg/controller"
	. "github.com/gardener/gardener/extensions/pkg/webhook/shoot"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	resourcesv1alpha1 "github.com/gardener/gardener/pkg/apis/resources/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	kubernetesutils "github.com/gardener/gardener/pkg/utils/kubernetes"
	. "github.com/gardener/gardener/pkg/utils/test/matchers"

	"github.com/hashicorp/go-multierror"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Webhook", func() {
	var (
		ctx        = context.TODO()
		fakeClient client.Client

		shootWebhookConfig    *admissionregistrationv1.MutatingWebhookConfiguration
		shootWebhookConfigRaw map[string][]byte

		extensionName       = "provider-test"
		managedResourceName = "extension-provider-test-shoot-webhooks"
		serverPort          = 1337
	)

	BeforeEach(func() {
		fakeClient = fakeclient.NewClientBuilder().WithScheme(kubernetes.SeedScheme).Build()

		shootWebhookConfig = &admissionregistrationv1.MutatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: extensionName,
			},
			Webhooks: []admissionregistrationv1.MutatingWebhook{{
				Name: "some-webhook",
			}},
		}
		shootWebhookConfigRaw = map[string][]byte{"mutatingwebhookconfiguration____provider-test.yaml": []byte(`apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  creationTimestamp: null
  name: provider-test
webhooks:
- admissionReviewVersions: null
  clientConfig: {}
  name: some-webhook
  sideEffects: null
`)}
	})

	Describe("#ReconcileWebhookConfig", func() {
		var (
			cluster   *controller.Cluster
			namespace = "extension-foo-bar"
		)

		BeforeEach(func() {
			cluster = &controller.Cluster{Shoot: &gardencorev1beta1.Shoot{}}
		})

		It("should reconcile the shoot webhook config", func() {
			Expect(ReconcileWebhookConfig(ctx, fakeClient, namespace, extensionName, managedResourceName, serverPort, shootWebhookConfig, cluster)).To(Succeed())
			expectWebhookConfigReconciliation(ctx, fakeClient, namespace, managedResourceName, shootWebhookConfigRaw)
		})
	})

	Describe("#ReconcileWebhooksForAllNamespaces", func() {
		var (
			networkPolicyName      = "gardener-extension-" + extensionName
			extensionType          = "test"
			shootNamespaceSelector = map[string]string{"networking.shoot.gardener.cloud/provider": extensionType}

			namespace1 *corev1.Namespace
			namespace2 *corev1.Namespace
			namespace3 *corev1.Namespace
			namespace4 *corev1.Namespace
			namespace5 *corev1.Namespace

			networkPolicy3 *networkingv1.NetworkPolicy
			networkPolicy4 *networkingv1.NetworkPolicy

			cluster3 *extensionsv1alpha1.Cluster
			cluster4 *extensionsv1alpha1.Cluster
		)

		BeforeEach(func() {
			namespace1 = &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "namespace1",
				},
			}
			namespace2 = &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "namespace2",
					Labels: map[string]string{
						"gardener.cloud/role":                      "shoot",
						"networking.shoot.gardener.cloud/provider": "foo",
					},
				},
			}
			namespace3 = &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "namespace3",
					Labels: map[string]string{
						"gardener.cloud/role":                      "shoot",
						"networking.shoot.gardener.cloud/provider": extensionType,
					},
				},
			}
			namespace4 = &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "namespace4",
					Labels: map[string]string{
						"gardener.cloud/role":                      "shoot",
						"networking.shoot.gardener.cloud/provider": extensionType,
					},
				},
			}
			namespace5 = &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "namespace5",
					Labels: map[string]string{
						"gardener.cloud/role":                      "shoot",
						"networking.shoot.gardener.cloud/provider": extensionType,
					},
				},
			}

			Expect(fakeClient.Create(ctx, namespace1)).To(Succeed())
			Expect(fakeClient.Create(ctx, namespace2)).To(Succeed())
			Expect(fakeClient.Create(ctx, namespace3)).To(Succeed())
			Expect(fakeClient.Create(ctx, namespace4)).To(Succeed())
			Expect(fakeClient.Create(ctx, namespace5)).To(Succeed())

			networkPolicy3 = &networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Namespace: namespace3.Name, Name: networkPolicyName}}
			networkPolicy4 = &networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Namespace: namespace4.Name, Name: networkPolicyName}}

			Expect(fakeClient.Create(ctx, networkPolicy3)).To(Succeed())
			Expect(fakeClient.Create(ctx, networkPolicy4)).To(Succeed())

			cluster3 = &extensionsv1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: namespace3.Name},
				Spec: extensionsv1alpha1.ClusterSpec{
					Shoot: runtime.RawExtension{
						Object: &gardencorev1beta1.Shoot{},
					},
				},
			}
			cluster4 = &extensionsv1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: namespace4.Name},
				Spec: extensionsv1alpha1.ClusterSpec{
					Shoot: runtime.RawExtension{
						Object: &gardencorev1beta1.Shoot{},
					},
				},
			}

			Expect(fakeClient.Create(ctx, cluster3)).To(Succeed())
			Expect(fakeClient.Create(ctx, cluster4)).To(Succeed())
		})

		It("should reconcile the webhook config for namespace3 and namespace4", func() {
			Expect(ReconcileWebhooksForAllNamespaces(ctx, fakeClient, extensionName, managedResourceName, shootNamespaceSelector, serverPort, shootWebhookConfig)).To(Succeed())

			expectNoWebhookConfigReconciliation(ctx, fakeClient, namespace1.Name, managedResourceName)
			expectNoWebhookConfigReconciliation(ctx, fakeClient, namespace2.Name, managedResourceName)
			expectWebhookConfigReconciliation(ctx, fakeClient, namespace3.Name, managedResourceName, shootWebhookConfigRaw)
			expectWebhookConfigReconciliation(ctx, fakeClient, namespace4.Name, managedResourceName, shootWebhookConfigRaw)
			expectNoWebhookConfigReconciliation(ctx, fakeClient, namespace5.Name, managedResourceName)
		})

		It("should return an error because cluster for namespace3 is missing", func() {
			Expect(fakeClient.Delete(ctx, cluster3)).To(Succeed())

			err := ReconcileWebhooksForAllNamespaces(ctx, fakeClient, extensionName, managedResourceName, shootNamespaceSelector, serverPort, shootWebhookConfig)

			Expect(err).To(BeAssignableToTypeOf(&multierror.Error{}))
			Expect(err.(*multierror.Error).Errors).To(ConsistOf(Equal(apierrors.NewNotFound(schema.GroupResource{Group: extensionsv1alpha1.SchemeGroupVersion.Group, Resource: "clusters"}, namespace3.Name))))
		})

		It("should return an error because cluster for namespace4 is does not contain shoot", func() {
			patch := client.MergeFrom(cluster4.DeepCopy())
			cluster4.Spec.Shoot = runtime.RawExtension{}
			Expect(fakeClient.Patch(ctx, cluster4, patch)).To(Succeed())

			err := ReconcileWebhooksForAllNamespaces(ctx, fakeClient, extensionName, managedResourceName, shootNamespaceSelector, serverPort, shootWebhookConfig)

			Expect(err).To(BeAssignableToTypeOf(&multierror.Error{}))
			Expect(err.(*multierror.Error).Errors).To(ConsistOf(Equal(errors.New("no shoot found in cluster resource"))))
		})
	})
})

func expectWebhookConfigReconciliation(ctx context.Context, fakeClient client.Client, namespace, managedResourceName string, shootWebhookConfigRaw map[string][]byte) {
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: managedResourceName, Namespace: namespace}}
	ExpectWithOffset(1, fakeClient.Get(ctx, client.ObjectKeyFromObject(secret), secret)).To(Succeed())
	ExpectWithOffset(1, secret.Type).To(Equal(corev1.SecretTypeOpaque))
	ExpectWithOffset(1, secret.Data).To(Equal(shootWebhookConfigRaw))

	managedResource := &resourcesv1alpha1.ManagedResource{ObjectMeta: metav1.ObjectMeta{Name: managedResourceName, Namespace: namespace}}
	ExpectWithOffset(1, fakeClient.Get(ctx, client.ObjectKeyFromObject(managedResource), managedResource)).To(Succeed())
	ExpectWithOffset(1, managedResource.Spec).To(DeepEqual(resourcesv1alpha1.ManagedResourceSpec{SecretRefs: []corev1.LocalObjectReference{{Name: managedResourceName}}}))
}

func expectNoWebhookConfigReconciliation(ctx context.Context, fakeClient client.Client, namespace, managedResourceName string) {
	ExpectWithOffset(1, fakeClient.Get(ctx, kubernetesutils.Key(namespace, managedResourceName), &corev1.Secret{})).To(BeNotFoundError())
	ExpectWithOffset(1, fakeClient.Get(ctx, kubernetesutils.Key(namespace, managedResourceName), &resourcesv1alpha1.ManagedResource{})).To(BeNotFoundError())
}
