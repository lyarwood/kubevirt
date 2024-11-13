package controller

import (
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
)

var (
	KeyFunc = cache.DeletionHandlingMetaNamespaceKeyFunc
)

func NamespacedKey(namespace, name string) string {
	return types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}.String()
}
