package process

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	virtv1 "kubevirt.io/api/core/v1"
	templatev1alpha1 "kubevirt.io/api/template/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	"sigs.k8s.io/yaml"

	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
)

const (
	PROCESS = "process"
)

type process struct {
	namespace string
	client    kubecli.KubevirtClient
}

func NewCommand() *cobra.Command {
	p := process{}
	cmd := &cobra.Command{
		Use:   PROCESS,
		Short: "Process a VirtualMachineTemplate into a VirtualMachine.",
		Run: func(cmd *cobra.Command, args []string) {
			if err := p.Process(cmd, args); err != nil {
				cmd.PrintErrln(err)
				os.Exit(1)
			}
		},
	}
	return cmd
}

func parseArgs(args []string) (string, []string, error) {
	name := ""
	params := []string{}
	for _, s := range args {
		isValue := strings.Contains(s, "=")
		switch {
		case isValue:
			params = append(params, s)
		case !isValue && len(name) == 0:
			name = s
		case !isValue && len(name) > 0:
			return name, params, fmt.Errorf("template name must be specified only once: %s", s)
		}
	}
	return name, params, nil
}

func (p process) Process(cmd *cobra.Command, args []string) error {
	name, params, err := parseArgs(args)
	if err != nil {
		return err
	}

	if name == "" {
		return fmt.Errorf("template name is required")
	}

	if p.client, p.namespace, _, err = clientconfig.ClientAndNamespaceFromContext(cmd.Context()); err != nil {
		return fmt.Errorf("cannot obtain KubeVirt client: %v", err)
	}

	template, err := p.client.GeneratedKubeVirtClient().TemplateV1alpha1().VirtualMachineTemplates(p.namespace).Get(cmd.Context(), name, v1.GetOptions{})
	if err != nil {
		return err
	}

	if errs := injectUserVars(params, template, true); errs != nil {
		var errMsgs []string
		for _, err := range errs {
			errMsgs = append(errMsgs, err.Error())
		}
		return fmt.Errorf("failed injecting params: %s", strings.Join(errMsgs, "; "))
	}

	processor := NewProcessor(map[string]Generator{
		"expression": NewExpressionValueGenerator(rand.New(rand.NewSource(time.Now().UnixNano()))),
	})

	if errs := processor.Process(template); len(errs) > 0 {
		var errMsgs []string
		for _, err := range errs {
			errMsgs = append(errMsgs, err.Error())
		}
		return fmt.Errorf("failed to process template: %s", strings.Join(errMsgs, "; "))
	}

	// Convert the runtime object to VirtualMachine
	vm := &virtv1.VirtualMachine{}

	// Try direct type assertion first
	if vmObj, ok := template.Spec.VirtualMachine.Object.(*virtv1.VirtualMachine); ok {
		vm = vmObj
	} else if len(template.Spec.VirtualMachine.Raw) > 0 {
		// If Raw bytes are available, use them
		if err := json.Unmarshal(template.Spec.VirtualMachine.Raw, vm); err != nil {
			return fmt.Errorf("failed to decode VirtualMachine from Raw bytes: %v", err)
		}
	} else if unstruct, ok := template.Spec.VirtualMachine.Object.(*unstructured.Unstructured); ok {
		// Convert unstructured object to VirtualMachine
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstruct.Object, vm); err != nil {
			return fmt.Errorf("failed to convert unstructured object to VirtualMachine: %v", err)
		}
	} else {
		return fmt.Errorf("unable to convert template VirtualMachine: Object is %T", template.Spec.VirtualMachine.Object)
	}

	out, err := yaml.Marshal(vm)
	if err != nil {
		return fmt.Errorf("failed to marshal VirtualMachine to YAML: %v", err)
	}
	cmd.Print(string(out))

	return nil
}

// injectUserVars injects user specified variables into the Template
func injectUserVars(params []string, t *templatev1alpha1.VirtualMachineTemplate, ignoreUnknownParameters bool) []error {
	var errors []error
	for _, param := range params {
		s := strings.SplitN(param, "=", 2)
		if len(s) != 2 {
			errors = append(errors, fmt.Errorf("invalid parameter format %q, expected key=value", param))
			continue
		}
		k := strings.TrimSpace(s[0])
		v := s[1] // Don't trim value as it might be intentionally have leading/trailing spaces

		if k == "" {
			errors = append(errors, fmt.Errorf("parameter key cannot be empty in %q", param))
			continue
		}

		p := GetParameterByName(t, k)
		if p != nil {
			p.Value = v
			p.Generate = ""
		} else if !ignoreUnknownParameters {
			errors = append(errors, fmt.Errorf("unknown parameter name %q", k))
		}
	}
	return errors
}
