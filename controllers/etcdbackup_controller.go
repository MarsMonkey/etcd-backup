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

package controllers

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	etcdv1alpha1 "github.com/MarsMonkey/etcd-backup/api/v1alpha1"
)

// EtcdBackupReconciler reconciles a EtcdBackup object
type EtcdBackupReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=etcd.douban.io,resources=etcdbackups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=etcd.douban.io,resources=etcdbackups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=etcd.douban.io,resources=etcdbackups/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the EtcdBackup object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *EtcdBackupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EtcdBackupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&etcdv1alpha1.EtcdBackup{}).
		Complete(r)
}

// backupState ?????? EtcdBackup ???????????????????????? (???????????????????????????status)
type backupState struct {
	backup *etcdv1alpha1.EtcdBackup // EtcdBackup ????????????
	actual *backupStateContainer // ????????????
	desired *backupStateContainer //????????????
}

// backupStateContainer contains the state of EtcdBackup
type backupStateContainer struct {
	pod *corev1.Pod
}

// setStateActual ???????????? backupState ???????????????
func (r *EtcdBackupReconciler ) setStateActual (ctx context.Context, state *backupState) error {
	var actual backupStateContainer

	key := client.ObjectKey {
		Name: 	state.backup.Name,
		Namespace: state.backup.Namespace,
	}

	// get pod
	actual.pod = &corev1.Pod{}
	if err := r.Get(ctx, key, actual.pod); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return fmt.Errorf("getting pod err: %s", err)
		}
		actual.pod = nil
	}

	// ???????????????????????????
	state.actual = &actual
	return nil
}

// setStateDesired ???????????? backupState ??????????????? (?????? EtcdBackup ??????)
func (r *EtcdBackupReconciler) setStateDesired(state *backupState) error {
	var desired backupStateContainer

	// ?????????????????????pod????????????????????????
	pod, err := podForBackup(state.backup, r.BackupAgentImage)
	if err != nil {
		return fmt.Errorf("computing pod for backup error: %q", err)
	}
	// controller reference
	if err := controllerutil.SetControllerReference(state.backup, pod, r.Scheme); err != nil {
		return fmt.Errorf("setting pod controller reference error: %s", err)
	}
	desired.pod = pod
	// ?????????????????????
	state.desired = &desired
	return nil
}

// getState ??????????????????????????????????????????????????????????????????????????????
func (r EtcdBackupReconciler) getState(ctx context.Context, req ctrl.Request) (*backupState, error) {
	var state backupState

	// ?????? EtcdBackup ??????
	state.backup = &etcdv1alpha1.EtcdBackup{}
	if err := r.Get(ctx, req.NamespacedName, state.backup); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return nil, fmt.Errorf("getting backup error: %s", err)
		}
		// ???????????????????????????
		state.backup = nil
		return &state, nil
	}

	// ?????????????????????????????????
	if err := r.setStateActual(ctx, &state); err != nil {
		return nil, fmt.Errorf("setting actual state error: %s", err)
	}

	// ???????????????????????????
	if err := r.setStateDesired(&state); err != nil {
		return nil, fmt.Errorf("setting desired state error: %s", err)
	}

	return &state, nil
}

// podForBackup ????????????pod??????????????????
func podForBackup(backup *etcdv1alpha1.EtcdBackup, image string) (*corev1.Pod, error) {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: backup.Name,
			Namespace: backup.Namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "backup-agent",
					Image: image,
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("50Mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("50Mi"),
						},
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}, nil
}