package tenantControlNamespace

import (
	"context"
	"strings"

	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"

	"github.com/liqotech/liqo/pkg/discovery"
)

// add the bindings for the remote clusterID for the given ClusterRoles
// This method creates RoleBindings in the Tenant Control Namespace for a remote identity
func (nm *tenantControlNamespaceManager) BindClusterRoles(clusterID string, clusterRoles ...*rbacv1.ClusterRole) ([]*rbacv1.RoleBinding, error) {
	namespace, err := nm.GetNamespace(clusterID)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	bindings := make([]*rbacv1.RoleBinding, len(clusterRoles))
	for i, clusterRole := range clusterRoles {
		bindings[i], err = nm.bindClusterRole(clusterID, namespace, clusterRole)
		if err != nil {
			klog.Error(err)
			return nil, err
		}
	}
	return bindings, nil
}

// remove the bindings for the remote clusterID for the given ClusterRoles
// This method deletes RoleBindings in the Tenant Control Namespace for a remote identity
func (nm *tenantControlNamespaceManager) UnbindClusterRoles(clusterID string, clusterRoles ...string) error {
	namespace, err := nm.GetNamespace(clusterID)
	if err != nil {
		klog.Error(err)
		return err
	}

	for _, clusterRole := range clusterRoles {
		if err = nm.unbindClusterRole(namespace, clusterRole); err != nil {
			klog.Error(err)
			return err
		}
	}
	return nil
}

// create a RoleBinding for the given clusterID in the given Namespace
func (nm *tenantControlNamespaceManager) bindClusterRole(clusterID string, namespace *v1.Namespace, clusterRole *rbacv1.ClusterRole) (*rbacv1.RoleBinding, error) {
	ownerRef := metav1.OwnerReference{
		APIVersion: rbacv1.SchemeGroupVersion.String(),
		Kind:       "ClusterRole",
		Name:       clusterRole.Name,
		UID:        clusterRole.UID,
	}

	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: strings.Join([]string{roleBindingRoot, clusterRole.Name, ""}, "-"),
			Namespace:    namespace.Name,
			Labels: map[string]string{
				discovery.ClusterRoleLabel: clusterRole.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				ownerRef,
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:     rbacv1.UserKind,
				APIGroup: rbacv1.GroupName,
				Name:     clusterID,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     clusterRole.Name,
		},
	}

	return nm.client.RbacV1().RoleBindings(namespace.Name).Create(context.TODO(), rb, metav1.CreateOptions{})
}

// delete a RoleBinding in the given Namespace
func (nm *tenantControlNamespaceManager) unbindClusterRole(namespace *v1.Namespace, clusterRole string) error {
	labelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			discovery.ClusterRoleLabel: clusterRole,
		},
	}

	return nm.client.RbacV1().RoleBindings(namespace.Name).DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	})
}