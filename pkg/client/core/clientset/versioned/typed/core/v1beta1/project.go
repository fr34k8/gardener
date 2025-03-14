// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

// Code generated by client-gen. DO NOT EDIT.

package v1beta1

import (
	context "context"

	corev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	scheme "github.com/gardener/gardener/pkg/client/core/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	gentype "k8s.io/client-go/gentype"
)

// ProjectsGetter has a method to return a ProjectInterface.
// A group's client should implement this interface.
type ProjectsGetter interface {
	Projects() ProjectInterface
}

// ProjectInterface has methods to work with Project resources.
type ProjectInterface interface {
	Create(ctx context.Context, project *corev1beta1.Project, opts v1.CreateOptions) (*corev1beta1.Project, error)
	Update(ctx context.Context, project *corev1beta1.Project, opts v1.UpdateOptions) (*corev1beta1.Project, error)
	// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
	UpdateStatus(ctx context.Context, project *corev1beta1.Project, opts v1.UpdateOptions) (*corev1beta1.Project, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*corev1beta1.Project, error)
	List(ctx context.Context, opts v1.ListOptions) (*corev1beta1.ProjectList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *corev1beta1.Project, err error)
	ProjectExpansion
}

// projects implements ProjectInterface
type projects struct {
	*gentype.ClientWithList[*corev1beta1.Project, *corev1beta1.ProjectList]
}

// newProjects returns a Projects
func newProjects(c *CoreV1beta1Client) *projects {
	return &projects{
		gentype.NewClientWithList[*corev1beta1.Project, *corev1beta1.ProjectList](
			"projects",
			c.RESTClient(),
			scheme.ParameterCodec,
			"",
			func() *corev1beta1.Project { return &corev1beta1.Project{} },
			func() *corev1beta1.ProjectList { return &corev1beta1.ProjectList{} },
		),
	}
}
