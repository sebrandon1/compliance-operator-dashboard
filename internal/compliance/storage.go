package compliance

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sebrandon1/compliance-operator-dashboard/internal/k8s"
)

const (
	hostpathCSIDriverName     = "kubevirt.io.hostpath-provisioner"
	defaultSCAnnotation       = "storageclass.kubernetes.io/is-default-class"
	crcCSIHostpathProvisioner = "crc-csi-hostpath-provisioner"
	localPathProvisioner      = "rancher.io/local-path"
	hostpathProvisionerName   = "kubevirt.io.hostpath-provisioner"
)

// DetectStorage checks the cluster for storage provisioners and default StorageClass.
// Reimplements the storage detection from install-compliance-operator.sh lines 43-90.
func DetectStorage(ctx context.Context, client *k8s.Client) (*StorageInfo, error) {
	if client == nil {
		return nil, fmt.Errorf("kubernetes client is nil")
	}

	info := &StorageInfo{}

	// Check if hostpath CSI driver is deployed
	_, err := client.Clientset.StorageV1().CSIDrivers().Get(ctx, hostpathCSIDriverName, metav1.GetOptions{})
	if err == nil {
		info.HostpathCSIDeployed = true
	}

	// Find default StorageClass
	storageClasses, err := client.Clientset.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return info, fmt.Errorf("listing storage classes: %w", err)
	}

	for _, sc := range storageClasses.Items {
		if sc.Annotations[defaultSCAnnotation] == "true" {
			info.HasDefaultStorageClass = true
			info.StorageClassName = sc.Name
			info.Provisioner = sc.Provisioner
			break
		}
	}

	// Build recommendation
	if info.HostpathCSIDeployed {
		info.Recommendation = "KubeVirt HostPath CSI driver detected (recommended)"
	} else if !info.HasDefaultStorageClass {
		info.Recommendation = "No default StorageClass detected. Consider deploying the HostPath CSI driver."
	} else if info.Provisioner == localPathProvisioner {
		info.Recommendation = "local-path provisioner detected. This may have permission issues with restricted-v2 SCC. Consider deploying the HostPath CSI driver."
	}

	// If no default found, try to find crc-csi-hostpath-provisioner
	if !info.HasDefaultStorageClass {
		for _, sc := range storageClasses.Items {
			if sc.Name == crcCSIHostpathProvisioner {
				info.StorageClassName = sc.Name
				info.Provisioner = sc.Provisioner
				break
			}
		}
		// Fall back to first available
		if info.StorageClassName == "" && len(storageClasses.Items) > 0 {
			info.StorageClassName = storageClasses.Items[0].Name
			info.Provisioner = storageClasses.Items[0].Provisioner
		}
	}

	return info, nil
}
