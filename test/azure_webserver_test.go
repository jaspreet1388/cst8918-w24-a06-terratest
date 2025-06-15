package test

import (
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/azure"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/gruntwork-io/terratest/modules/retry"
	"github.com/stretchr/testify/assert"
)

var subscriptionID string = "1dcb97e8-a9e3-477e-ab93-25569869c665"

func TestAzureLinuxVMCreation(t *testing.T) {
	terraformOptions := &terraform.Options{
		TerraformDir: "../",
		Vars: map[string]interface{}{
			"labelPrefix": "chha0038",
		},
	}

	// Retry destroy on transient NIC update issues
	defer func() {
		_, err := retry.DoWithRetryE(t, "Destroy with retry", 3, 30*time.Second, func() (string, error) {
			_, err := terraform.DestroyE(t, terraformOptions)
			if err != nil {
				t.Logf("Retrying destroy due to error: %v", err)
			}
			return "", err
		})
		assert.NoError(t, err, "Terraform destroy should eventually succeed")
	}()

	// Run terraform init and apply
	terraform.InitAndApply(t, terraformOptions)

	// Retrieve outputs from Terraform
	vmName := terraform.Output(t, terraformOptions, "vm_name")
	resourceGroupName := terraform.Output(t, terraformOptions, "resource_group_name")
	nicName := terraform.Output(t, terraformOptions, "nic_name")

	// 1. Confirm VM exists
	assert.True(t, azure.VirtualMachineExists(t, vmName, resourceGroupName, subscriptionID), "VM should exist")

	// 2. Confirm NIC exists and is connected to VM
	assert.True(t, azure.NetworkInterfaceExists(t, nicName, resourceGroupName, subscriptionID), "NIC should exist")
	nic, err := azure.GetNetworkInterfaceE(nicName, resourceGroupName, subscriptionID)
	assert.NoError(t, err)
	assert.NotNil(t, nic.VirtualMachine, "NIC should be attached to a VM")
	assert.Contains(t, *nic.VirtualMachine.ID, vmName, "NIC should be attached to the correct VM")

	// 3. Confirm correct Ubuntu version
	vm := azure.GetVirtualMachine(t, vmName, resourceGroupName, subscriptionID)
	image := vm.StorageProfile.ImageReference
	// Log details for visibility
	t.Logf("VM Image Publisher: %s", *image.Publisher)
	t.Logf("VM Image Offer: %s", *image.Offer)
	t.Logf("VM Image SKU: %s", *image.Sku)
	assert.Equal(t, "Canonical", *image.Publisher, "VM image publisher should be Canonical")
	assert.Equal(t, "0001-com-ubuntu-server-jammy", *image.Offer, "VM offer should be Ubuntu Jammy")
	assert.True(t, strings.Contains(*image.Sku, "22_04"), "VM SKU should be Ubuntu 22.04")
}

