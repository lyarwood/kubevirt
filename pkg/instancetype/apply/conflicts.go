package apply

import (
	"strings"

	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
)

type Conflicts []*k8sfield.Path

func (c Conflicts) String() string {
	pathStrings := make([]string, 0, len(c))
	for _, path := range c {
		pathStrings = append(pathStrings, path.String())
	}
	return strings.Join(pathStrings, ", ")
}
