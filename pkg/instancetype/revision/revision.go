package revision

import (
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/client-go/kubecli"
)

type Revision struct {
	instancetypeStore        cache.Store
	clusterInstancetypeStore cache.Store
	preferenceStore          cache.Store
	clusterPreferenceStore   cache.Store
	controllerRevisionStore  cache.Store
	virtClient               kubecli.KubevirtClient
}

func NewRevision() *Revision {
	return &Revision{}
}
