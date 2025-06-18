/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package template

import (
	"fmt"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/yaml"

	templatev1alpha1 "kubevirt.io/api/template/v1alpha1"

	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	NameFlag = "name"

	randSuffixLength = 5
)

type createTemplate struct {
	name      string
	vmName    string
	namespace string

	cmd *cobra.Command
}

func NewCommand() *cobra.Command {
	c := defaultCreateTemplate()
	cmd := &cobra.Command{
		Use:   "template VM_NAME",
		Short: "Create a VirtualMachineTemplateRequest manifest.",
		Long: "Create a VirtualMachineTemplateRequest manifest.\n\n" +
			"The VM_NAME argument specifies the source VirtualMachine to create a template from.",
		Args:    cobra.ExactArgs(1),
		Example: c.usage(),
		RunE:    c.run,
	}

	cmd.Flags().StringVar(&c.name, NameFlag, c.name, "Specify the name of the VirtualMachineTemplateRequest.")

	cmd.Flags().SortFlags = false
	cmd.SetUsageTemplate(templates.UsageTemplate())

	return cmd
}

func defaultCreateTemplate() createTemplate {
	return createTemplate{}
}

func (c *createTemplate) run(cmd *cobra.Command, args []string) error {
	if err := c.setDefaults(cmd, args); err != nil {
		return err
	}

	templateRequest, err := c.newTemplateRequest()
	if err != nil {
		return err
	}

	out, err := yaml.Marshal(templateRequest)
	if err != nil {
		return err
	}

	cmd.Print(string(out))

	return nil
}

func (c *createTemplate) setDefaults(cmd *cobra.Command, args []string) error {
	c.cmd = cmd
	c.vmName = args[0]

	_, namespace, overridden, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return err
	}
	if overridden {
		c.namespace = namespace
	}

	if c.name == "" {
		c.name = fmt.Sprintf("template-request-%s", rand.String(randSuffixLength))
	}

	return nil
}

func (c *createTemplate) newTemplateRequest() (*templatev1alpha1.VirtualMachineTemplateRequest, error) {
	templateRequest := &templatev1alpha1.VirtualMachineTemplateRequest{
		TypeMeta: metav1.TypeMeta{
			Kind:       "VirtualMachineTemplateRequest",
			APIVersion: templatev1alpha1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: c.name,
		},
		Spec: templatev1alpha1.VirtualMachineTemplateRequestSpec{
			Source: templatev1alpha1.VirtualMachineReference{
				Name: c.vmName,
			},
		},
	}

	if c.namespace != "" {
		templateRequest.Namespace = c.namespace
	}

	return templateRequest, nil
}

func (c *createTemplate) usage() string {
	return `  # Create a manifest for a VirtualMachineTemplateRequest with a random name:
  {{ProgramName}} create template my-vm

  # Create a manifest for a VirtualMachineTemplateRequest with a specified name:
  {{ProgramName}} create template my-vm --name=my-template-request`
}
