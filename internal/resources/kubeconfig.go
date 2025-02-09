// Copyright 2022 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kamajiv1alpha1 "github.com/clastix/kamaji/api/v1alpha1"
	"github.com/clastix/kamaji/internal/constants"
	"github.com/clastix/kamaji/internal/kubeadm"
	"github.com/clastix/kamaji/internal/utilities"
)

const (
	AdminKubeConfigFileName             = kubeadmconstants.AdminKubeConfigFileName
	ControllerManagerKubeConfigFileName = kubeadmconstants.ControllerManagerKubeConfigFileName
	SchedulerKubeConfigFileName         = kubeadmconstants.SchedulerKubeConfigFileName
	localhost                           = "127.0.0.1"
)

type KubeconfigResource struct {
	resource           *corev1.Secret
	Client             client.Client
	Name               string
	KubeConfigFileName string
	TmpDirectory       string
}

func (r *KubeconfigResource) ShouldStatusBeUpdated(context.Context, *kamajiv1alpha1.TenantControlPlane) bool {
	return false
}

func (r *KubeconfigResource) ShouldCleanup(*kamajiv1alpha1.TenantControlPlane) bool {
	return false
}

func (r *KubeconfigResource) CleanUp(context.Context, *kamajiv1alpha1.TenantControlPlane) (bool, error) {
	return false, nil
}

func (r *KubeconfigResource) Define(ctx context.Context, tenantControlPlane *kamajiv1alpha1.TenantControlPlane) error {
	r.resource = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.getPrefixedName(tenantControlPlane),
			Namespace: tenantControlPlane.GetNamespace(),
		},
	}

	return nil
}

func (r *KubeconfigResource) getPrefixedName(tenantControlPlane *kamajiv1alpha1.TenantControlPlane) string {
	return utilities.AddTenantPrefix(r.GetName(), tenantControlPlane)
}

func (r *KubeconfigResource) GetClient() client.Client {
	return r.Client
}

func (r *KubeconfigResource) GetTmpDirectory() string {
	return r.TmpDirectory
}

func (r *KubeconfigResource) GetName() string {
	return r.Name
}

func (r *KubeconfigResource) UpdateTenantControlPlaneStatus(ctx context.Context, tenantControlPlane *kamajiv1alpha1.TenantControlPlane) error {
	logger := log.FromContext(ctx, "resource", r.GetName())

	status, err := r.getKubeconfigStatus(tenantControlPlane)
	if err != nil {
		logger.Error(err, "cannot retrieve status")

		return err
	}

	status.LastUpdate = metav1.Now()
	status.SecretName = r.resource.GetName()
	status.Checksum = utilities.GetObjectChecksum(r.resource)

	return nil
}

func (r *KubeconfigResource) getKubeconfigStatus(tenantControlPlane *kamajiv1alpha1.TenantControlPlane) (*kamajiv1alpha1.KubeconfigStatus, error) {
	switch r.KubeConfigFileName {
	case kubeadmconstants.AdminKubeConfigFileName:
		return &tenantControlPlane.Status.KubeConfig.Admin, nil
	case kubeadmconstants.ControllerManagerKubeConfigFileName:
		return &tenantControlPlane.Status.KubeConfig.ControllerManager, nil
	case kubeadmconstants.SchedulerKubeConfigFileName:
		return &tenantControlPlane.Status.KubeConfig.Scheduler, nil
	default:
		return nil, fmt.Errorf("kubeconfigfilename %s is not a right name", r.KubeConfigFileName)
	}
}

func (r *KubeconfigResource) CreateOrUpdate(ctx context.Context, tenantControlPlane *kamajiv1alpha1.TenantControlPlane) (controllerutil.OperationResult, error) {
	return utilities.CreateOrUpdateWithConflict(ctx, r.Client, r.resource, r.mutate(ctx, tenantControlPlane))
}

func (r *KubeconfigResource) checksum(apiServerCertificatesSecret *corev1.Secret, kubeadmChecksum string) string {
	return utilities.CalculateMapChecksum(map[string][]byte{
		"ca-cert-checksum": apiServerCertificatesSecret.Data[kubeadmconstants.CACertName],
		"ca-key-checksum":  apiServerCertificatesSecret.Data[kubeadmconstants.CAKeyName],
		"kubeadmconfig":    []byte(kubeadmChecksum),
	})
}

func (r *KubeconfigResource) mutate(ctx context.Context, tenantControlPlane *kamajiv1alpha1.TenantControlPlane) controllerutil.MutateFn {
	return func() error {
		logger := log.FromContext(ctx, "resource", r.GetName())

		config, err := getStoredKubeadmConfiguration(ctx, r.Client, r.TmpDirectory, tenantControlPlane)
		if err != nil {
			logger.Error(err, "cannot retrieve kubeadm configuration")

			return err
		}

		if err = r.customizeConfig(config); err != nil {
			logger.Error(err, "cannot customize the configuration")

			return err
		}

		apiServerCertificatesSecretNamespacedName := k8stypes.NamespacedName{Namespace: tenantControlPlane.GetNamespace(), Name: tenantControlPlane.Status.Certificates.CA.SecretName}
		apiServerCertificatesSecret := &corev1.Secret{}
		if err := r.Client.Get(ctx, apiServerCertificatesSecretNamespacedName, apiServerCertificatesSecret); err != nil {
			logger.Error(err, "cannot retrieve the CA")

			return err
		}

		checksum := r.checksum(apiServerCertificatesSecret, config.Checksum())

		status, err := r.getKubeconfigStatus(tenantControlPlane)
		if err != nil {
			logger.Error(err, "cannot retrieve status")

			return err
		}

		if status.Checksum == checksum && kubeadm.IsKubeconfigValid(r.resource.Data[r.KubeConfigFileName]) {
			return nil
		}

		kubeconfig, err := kubeadm.CreateKubeconfig(
			r.KubeConfigFileName,

			kubeadm.CertificatePrivateKeyPair{
				Certificate: apiServerCertificatesSecret.Data[kubeadmconstants.CACertName],
				PrivateKey:  apiServerCertificatesSecret.Data[kubeadmconstants.CAKeyName],
			},
			config,
		)
		if err != nil {
			logger.Error(err, "cannot create a valid kubeconfig")

			return err
		}
		r.resource.Data = map[string][]byte{
			r.KubeConfigFileName: kubeconfig,
		}

		r.resource.SetLabels(utilities.KamajiLabels(tenantControlPlane.GetName(), r.GetName()))

		r.resource.SetAnnotations(map[string]string{
			constants.Checksum: checksum,
		})

		return ctrl.SetControllerReference(tenantControlPlane, r.resource, r.Client.Scheme())
	}
}

func (r *KubeconfigResource) customizeConfig(config *kubeadm.Configuration) error {
	switch r.KubeConfigFileName {
	case kubeadmconstants.ControllerManagerKubeConfigFileName:
		return r.localhostAsAdvertiseAddress(config)
	case kubeadmconstants.SchedulerKubeConfigFileName:
		return r.localhostAsAdvertiseAddress(config)
	default:
		return nil
	}
}

func (r *KubeconfigResource) localhostAsAdvertiseAddress(config *kubeadm.Configuration) error {
	config.InitConfiguration.LocalAPIEndpoint.AdvertiseAddress = localhost

	return nil
}
