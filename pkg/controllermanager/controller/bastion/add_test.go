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

package bastion_test

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/gardener/gardener/pkg/apis/operations"
	operationsv1alpha1 "github.com/gardener/gardener/pkg/apis/operations/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	. "github.com/gardener/gardener/pkg/controllermanager/controller/bastion"
	bastionstrategy "github.com/gardener/gardener/pkg/registry/operations/bastion"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Add", func() {
	Describe("ShootPredicate", func() {
		var obj *gardencorev1beta1.Shoot

		BeforeEach(func() {
			obj = &gardencorev1beta1.Shoot{}
		})

		Describe("#Create", func() {
			var e event.CreateEvent

			BeforeEach(func() {
				e = event.CreateEvent{Object: obj}
			})

			It("should return false if the object is not deleting", func() {
				Expect(ShootPredicate.Create(e)).To(BeFalse())
			})

			It("should return true if object is deleting", func() {
				obj.DeletionTimestamp = &metav1.Time{}
				Expect(ShootPredicate.Create(e)).To(BeTrue())
			})
		})

		Describe("#Delete", func() {
			var e event.DeleteEvent

			BeforeEach(func() {
				e = event.DeleteEvent{Object: obj}
			})

			It("should return false if the object is not deleting", func() {
				Expect(ShootPredicate.Delete(e)).To(BeFalse())
			})

			It("should return true if object is deleting", func() {
				obj.DeletionTimestamp = &metav1.Time{}
				Expect(ShootPredicate.Delete(e)).To(BeTrue())
			})
		})

		Describe("#Generic", func() {
			var e event.GenericEvent

			BeforeEach(func() {
				e = event.GenericEvent{Object: obj}
			})

			It("should return false if the object is not deleting", func() {
				Expect(ShootPredicate.Generic(e)).To(BeFalse())
			})

			It("should return true if object is deleting", func() {
				obj.DeletionTimestamp = &metav1.Time{}
				Expect(ShootPredicate.Generic(e)).To(BeTrue())
			})
		})

		Describe("#Update", func() {
			var (
				e      event.UpdateEvent
				objNew *gardencorev1beta1.Shoot
			)

			BeforeEach(func() {
				objNew = obj.DeepCopy()
				e = event.UpdateEvent{ObjectOld: obj, ObjectNew: objNew}
			})

			It("should return false if the object is not deleting and seed name did not change", func() {
				Expect(ShootPredicate.Update(e)).To(BeFalse())
			})

			It("should return false when shoot is scheduled for the first time", func() {
				obj.Spec.SeedName = nil
				objNew.Spec.SeedName = pointer.String("some-seed-name")

				Expect(ShootPredicate.Update(e)).To(BeFalse())
			})

			It("should return true when seed name changed", func() {
				obj.Spec.SeedName = pointer.String("old-seed")
				objNew.Spec.SeedName = pointer.String("new-seed")

				Expect(ShootPredicate.Update(e)).To(BeTrue())
			})

			It("should return true if object is deleting", func() {
				objNew.DeletionTimestamp = &metav1.Time{}
				Expect(ShootPredicate.Update(e)).To(BeTrue())
			})
		})
	})

	Describe("MapShootToBastions", func() {
		var (
			ctx        = context.TODO()
			log        logr.Logger
			fakeClient client.Client
		)

		BeforeEach(func() {
			log = logr.Discard()
			fakeClient = clientWithFieldSelectorSupport{fakeclient.NewClientBuilder().WithScheme(kubernetes.GardenScheme).Build()}
		})

		It("should do nothing if the object is no shoot", func() {
			Expect(MapShootToBastions(ctx, log, fakeClient, &corev1.Secret{})).To(BeEmpty())
		})

		It("should map the shoot to bastions", func() {
			var (
				shoot = &gardencorev1beta1.Shoot{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "some-namespace",
					},
				}
				bastion1 = &operationsv1alpha1.Bastion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bastion1",
						Namespace: shoot.Namespace,
					},
					Spec: operationsv1alpha1.BastionSpec{
						ShootRef: corev1.LocalObjectReference{
							Name: shoot.Name,
						},
					},
				}
				bastion2 = &operationsv1alpha1.Bastion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bastion2",
						Namespace: shoot.Namespace,
					},
					Spec: operationsv1alpha1.BastionSpec{
						ShootRef: corev1.LocalObjectReference{
							// the fake client does not implement the field selector options, so we should better use
							// the same shoot name here (otherwise, we could have tested with a different shoot name)
							Name: shoot.Name,
						},
					},
				}
				bastion3 = &operationsv1alpha1.Bastion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bastion3",
						Namespace: shoot.Namespace,
					},
					Spec: operationsv1alpha1.BastionSpec{
						ShootRef: corev1.LocalObjectReference{
							Name: "other",
						},
					},
				}
			)

			Expect(fakeClient.Create(ctx, bastion1)).To(Succeed())
			Expect(fakeClient.Create(ctx, bastion2)).To(Succeed())
			Expect(fakeClient.Create(ctx, bastion3)).To(Succeed())

			Expect(MapShootToBastions(ctx, log, fakeClient, shoot)).To(ConsistOf(
				reconcile.Request{NamespacedName: types.NamespacedName{Name: bastion1.Name, Namespace: bastion1.Namespace}},
				reconcile.Request{NamespacedName: types.NamespacedName{Name: bastion2.Name, Namespace: bastion2.Namespace}},
			))
		})
	})
})

// TODO: remove this again once the controller-runtime fake client supports field selectors
type clientWithFieldSelectorSupport struct {
	client.Client
}

func (c clientWithFieldSelectorSupport) List(ctx context.Context, obj client.ObjectList, opts ...client.ListOption) error {
	if err := c.Client.List(ctx, obj, opts...); err != nil {
		return err
	}

	listOpts := client.ListOptions{}
	listOpts.ApplyOptions(opts)

	if listOpts.FieldSelector != nil {
		objs, err := meta.ExtractList(obj)
		if err != nil {
			return err
		}
		filteredObjs, err := filterWithFieldSelector(objs, listOpts.FieldSelector)
		if err != nil {
			return err
		}
		err = meta.SetList(obj, filteredObjs)
		if err != nil {
			return err
		}
	}

	return nil
}

func filterWithFieldSelector(objs []runtime.Object, sel fields.Selector) ([]runtime.Object, error) {
	outItems := make([]runtime.Object, 0, len(objs))
	for _, obj := range objs {
		// convert to internal
		bastion := &operations.Bastion{}
		if err := kubernetes.GardenScheme.Convert(obj, bastion, nil); err != nil {
			return nil, err
		}

		fieldSet := bastionstrategy.ToSelectableFields(bastion)

		// complain about non-selectable fields if any
		for _, req := range sel.Requirements() {
			if !fieldSet.Has(req.Field) {
				return nil, fmt.Errorf("field selector not supported for field %q", req.Field)
			}
		}

		if !sel.Matches(fieldSet) {
			continue
		}
		outItems = append(outItems, obj.DeepCopyObject())
	}
	return outItems, nil
}