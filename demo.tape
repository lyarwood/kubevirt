# Demo: VirtualMachineTemplates and VirtualMachineTemplateRequests
# This demo shows the complete workflow from template processing to template creation

# Set up the recording environment
Set Theme "Catppuccin Mocha"
Set FontSize 14
Set Width 1200
Set Height 800
Set Padding 20
Set Margin 40
Set MarginFill "#1e1e2e"
Set BorderRadius 8

# Start the demo
Type "# 🚀 KubeVirt VirtualMachineTemplates and VirtualMachineTemplateRequests Demo"
Enter
Sleep 1s

Type "# This demo will show:"
Enter
Type "# 1. Processing an existing template to create a VM"
Enter
Type "# 2. Creating a new template from that VM using VirtualMachineTemplateRequest"
Enter
Type "# 3. Processing the new template to create another VM"
Enter
Sleep 2s

# Show the initial template
Type "# 📋 Step 1: Examining the initial VirtualMachineTemplate"
Enter
Sleep 1s

Type "yq template.yml"
Enter
Sleep 3s

Type ""
Enter
Type "# This template has:"
Enter
Type "# - Parameterized VM name (${NAME})"
Enter
Type "# - Configurable instance type (${INSTANCE_TYPE})"
Enter
Type "# - Fedora version parameter (${FEDORA_VERSION})"
Enter
Type "# - Container disk source for Fedora"
Enter
Sleep 3s

# Process the template to create a VM
Type "# 🔄 Step 2: Processing the template to create a VM with u1.medium instance type"
Enter
Sleep 1s

Type "kubectl apply -f template.yml"
Enter
Sleep 2s
Type "virtctl process fedora INSTANCE_TYPE=u1.medium FEDORA_VERSION=41 NAME=fedora | kubectl apply -f -"
Enter
Type "kubectl wait vms/fedora --for=condition=Ready"
Enter

Type "# ✅ VM created successfully!"
Enter
Sleep 1s

# Show the created VM
Type "# 📊 Step 3: Examining the created VM"
Enter
Sleep 1s

Type "kubectl get vm -o wide"
Enter
Sleep 2s

Type ""
Enter
Type "# Let us check the VM details:"
Enter
Sleep 1s

Type "kubectl get vms/fedora -o yaml | yq ."
Enter
Sleep 10s

Type ""
Enter
Type "# Notice how the template parameters were resolved:"
Enter
Type "# - NAME: User provided name of fedora used"
Enter
Type "# - INSTANCE_TYPE: Set to u1.medium as specified"
Enter
Type "# - FEDORA_VERSION: Used user provided value of 41"
Enter
Sleep 3s

# Wait for VM to be ready
Type "# ⏳ Step 4: Waiting for VM to be ready..."
Enter
Sleep 1s

Type "kubectl wait --for=condition=Ready vm --all --timeout=300s"
Enter
Sleep 2s

Type "# ✅ VM is ready!"
Enter
Sleep 1s

# Create VirtualMachineTemplateRequest
Type "# 📝 Step 5: Creating a VirtualMachineTemplateRequest to generate a new template"
Enter
Sleep 1s

Type "cat << 'EOF' > template-request.yml"
Enter
Type "apiVersion: template.kubevirt.io/v1alpha1"
Enter
Type "kind: VirtualMachineTemplateRequest"
Enter
Type "metadata:"
Enter
Type "  name: create-fedora-template"
Enter
Type "  namespace: default"
Enter
Type "spec:"
Enter
Type "  source:"
Enter
Type "    name: fedora
Enter
Type "EOF"
Enter
Sleep 2s

Type "cat template-request.yml"
Enter
Sleep 3s

Type ""
Enter
Type "# This request will:"
Enter
Type "# 1. Create a VirtualMachineSnapshot of the VM"
Enter
Type "# 2. Rewrite DataVolumeTemplates to use VolumeSnapshots"
Enter
Type "# 3. Generate a new VirtualMachineTemplate"
Enter
Sleep 3s

# Apply the template request
Type "# 🚀 Step 6: Applying the VirtualMachineTemplateRequest"
Enter
Sleep 1s

Type "kubectl apply -f template-request.yml"
Enter
Sleep 2s

Type "# ✅ Template request created!"
Enter
Sleep 1s

# Monitor the template request status
Type "# 📊 Step 7: Monitoring the template request status"
Enter
Sleep 1s

Type "kubectl get virtualmachinetemplaterequest create-fedora-template -o yaml"
Enter
Sleep 3s

Type ""
Enter
Type "# Let us watch the status as it progresses:"
Enter
Sleep 1s

Type "kubectl get virtualmachinetemplaterequest create-fedora-template -o jsonpath='{.status.phase}'"
Enter
Sleep 1s

Type ""
Enter
Type "# Checking conditions:"
Enter
Sleep 1s

Type "kubectl get virtualmachinetemplaterequest create-fedora-template -o jsonpath='{.status.conditions[*].type}: {.status.conditions[*].status}'"
Enter
Sleep 2s

# Wait for the template request to complete
Type "# ⏳ Step 8: Waiting for template request to complete..."
Enter
Sleep 1s

Type "kubectl wait --for=condition=Ready virtualmachinetemplaterequest create-fedora-template --timeout=300s"
Enter
Sleep 2s

Type "# ✅ Template request completed successfully!"
Enter
Sleep 1s

# Show the created template
Type "# 📋 Step 9: Examining the created VirtualMachineTemplate"
Enter
Sleep 1s

Type "kubectl get virtualmachinetemplate create-fedora-template-template -o yaml"
Enter
Sleep 4s

Type ""
Enter
Type "# Key differences in the new template:"
Enter
Type "# - DataVolumeTemplates now use VolumeSnapshots as sources"
Enter
Type "# - Names are prefixed with ${VM_NAME} to avoid conflicts"
Enter
Type "# - Template includes the exact VM state from the snapshot"
Enter
Sleep 3s

# Show the snapshot that was created
Type "# 📸 Step 10: Examining the created VirtualMachineSnapshot"
Enter
Sleep 1s

Type "kubectl get virtualmachinesnapshot create-fedora-template-snapshot -o yaml | head -20"
Enter
Sleep 3s

Type ""
Enter
Type "# The snapshot contains:"
Enter
Type "# - Complete VM specification"
Enter
Type "# - Volume mappings to VolumeSnapshots"
Enter
Type "# - Ready state for template creation"
Enter
Sleep 3s

# Process the new template to create another VM
Type "# 🔄 Step 11: Processing the new template to create another VM"
Enter
Sleep 1s

Type "virtctl process create-fedora-template-template --param VM_NAME=my-custom-vm | kubectl apply -f -"
Enter
Sleep 2s

Type "# ✅ Second VM created from the new template!"
Enter
Sleep 1s

# Show both VMs
Type "# 📊 Step 12: Comparing both VMs"
Enter
Sleep 1s

Type "kubectl get vm -o wide"
Enter
Sleep 2s

Type ""
Enter
Type "# Notice the naming pattern:"
Enter
Type "# - First VM: fedora-[generated-id] (from original template)"
Enter
Type "# - Second VM: my-custom-vm (from new template with custom parameter)"
Enter
Sleep 3s

# Show the DataVolumes
Type "# 💾 Step 13: Examining the DataVolumes"
Enter
Sleep 1s

Type "kubectl get datavolume -o wide"
Enter
Sleep 2s

Type ""
Enter
Type "# DataVolume naming shows the conflict-free approach:"
Enter
Type "# - Original: fedora (from container disk)"
Enter
Type "# - New template: my-custom-vm-fedora (prefixed with VM_NAME)"
Enter
Sleep 3s

# Show the VolumeSnapshots
Type "# 📸 Step 14: Examining the VolumeSnapshots"
Enter
Sleep 1s

Type "kubectl get volumesnapshot -o wide"
Enter
Sleep 2s

Type ""
Enter
Type "# VolumeSnapshots provide the data backing for the new template"
Enter
Sleep 2s

# Final summary
Type "# 🎉 Demo Complete!"
Enter
Sleep 1s

Type ""
Enter
Type "# What we accomplished:"
Enter
Type "# ✅ Processed an existing template to create a VM"
Enter
Type "# ✅ Created a VirtualMachineTemplateRequest"
Enter
Type "# ✅ Generated a new template with snapshot-backed DataVolumes"
Enter
Type "# ✅ Processed the new template to create another VM"
Enter
Type "# ✅ Demonstrated conflict-free naming with VM_NAME prefixing"
Enter
Sleep 3s

Type ""
Enter
Type "# Key benefits demonstrated:"
Enter
Type "# 🚀 Declarative template creation from existing VMs"
Enter
Type "# 💾 Snapshot-backed templates for consistent data state"
Enter
Type "# 🔧 Parameterized templates for customization"
Enter
Type "# 🛡️ Conflict-free naming for multiple VM instances"
Enter
Sleep 3s

Type ""
Enter
Type "# Thank you for watching the KubeVirt VirtualMachineTemplates demo! 🎬"
Enter
Sleep 2s

# Cleanup (hidden)
Hide
Type "# Cleaning up demo resources..."
Enter
Type "kubectl delete virtualmachinetemplaterequest create-fedora-template --ignore-not-found"
Enter
Type "kubectl delete virtualmachinetemplate create-fedora-template-template --ignore-not-found"
Enter
Type "kubectl delete vm --all --ignore-not-found"
Enter
Type "kubectl delete datavolume --all --ignore-not-found"
Enter
Type "kubectl delete virtualmachinesnapshot --all --ignore-not-found"
Enter
Type "kubectl delete volumesnapshot --all --ignore-not-found"
Enter
Type "rm -f template-request.yml"
Enter
Type "# Cleanup complete!"
Enter

Output demo.gif 