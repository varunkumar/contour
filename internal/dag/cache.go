// Copyright © 2018 Heptio
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dag

import (
	"sync"

	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"

	ingressroutev1 "github.com/heptio/contour/apis/contour/v1beta1"
)

const DEFAULT_INGRESS_CLASS = "contour"

// A KubernetesCache holds Kubernetes objects and associated configuration and produces
// DAG values.
type KubernetesCache struct {
	// IngressRouteRootNamespaces specifies the namespaces where root
	// IngressRoutes can be defined. If empty, roots can be defined in any
	// namespace.
	IngressRouteRootNamespaces []string

	// Contour's IngressClass.
	// If not set, defaults to DEFAULT_INGRESS_CLASS.
	IngressClass string

	sync.RWMutex

	ingresses     map[Meta]*v1beta1.Ingress
	ingressroutes map[Meta]*ingressroutev1.IngressRoute
	secrets       map[Meta]*v1.Secret
	delegations   map[Meta]*ingressroutev1.TLSCertificateDelegation
	services      map[Meta]*v1.Service
}

// Meta holds the name and namespace of a Kubernetes object.
type Meta struct {
	name, namespace string
}

// Insert inserts obj into the KubernetesCache.
// Insert returns true if the cache accepted the object, or false if the value
// is not interesting to the cache. If an object with a matching type, name,
// and namespace exists, it will be overwritten.
func (kc *KubernetesCache) Insert(obj interface{}) bool {
	switch obj := obj.(type) {
	case *v1.Secret:
		m := Meta{name: obj.Name, namespace: obj.Namespace}
		if kc.secrets == nil {
			kc.secrets = make(map[Meta]*v1.Secret)
		}
		kc.secrets[m] = obj
		return true
	case *v1.Service:
		m := Meta{name: obj.Name, namespace: obj.Namespace}
		if kc.services == nil {
			kc.services = make(map[Meta]*v1.Service)
		}
		kc.services[m] = obj
		return true
	case *v1beta1.Ingress:
		class := getIngressClassAnnotation(obj.Annotations)
		if class != "" && class != kc.ingressClass() {
			return false
		}
		m := Meta{name: obj.Name, namespace: obj.Namespace}
		if kc.ingresses == nil {
			kc.ingresses = make(map[Meta]*v1beta1.Ingress)
		}
		kc.ingresses[m] = obj
		return true
	case *ingressroutev1.IngressRoute:
		class := getIngressClassAnnotation(obj.Annotations)
		if class != "" && class != kc.ingressClass() {
			return false
		}
		m := Meta{name: obj.Name, namespace: obj.Namespace}
		if kc.ingressroutes == nil {
			kc.ingressroutes = make(map[Meta]*ingressroutev1.IngressRoute)
		}
		kc.ingressroutes[m] = obj
		return true
	case *ingressroutev1.TLSCertificateDelegation:
		m := Meta{name: obj.Name, namespace: obj.Namespace}
		if kc.delegations == nil {
			kc.delegations = make(map[Meta]*ingressroutev1.TLSCertificateDelegation)
		}
		kc.delegations[m] = obj
		return true
	default:
		// not an interesting object
		return false
	}
}

// ingressClass returns the IngressClass
// or DEFAULT_INGRESS_CLASS if not configured.
func (kc *KubernetesCache) ingressClass() string {
	return stringOrDefault(kc.IngressClass, DEFAULT_INGRESS_CLASS)
}

// Remove removes obj from the KubernetesCache.
// Remove returns a boolean indiciating if the cache changed after the remove operation.
func (kc *KubernetesCache) Remove(obj interface{}) bool {
	switch obj := obj.(type) {
	default:
		return kc.remove(obj)
	case cache.DeletedFinalStateUnknown:
		return kc.Remove(obj.Obj) // recurse into ourselves with the tombstoned value
	}
}

func (kc *KubernetesCache) remove(obj interface{}) bool {
	switch obj := obj.(type) {
	case *v1.Secret:
		m := Meta{name: obj.Name, namespace: obj.Namespace}
		_, ok := kc.secrets[m]
		delete(kc.secrets, m)
		return ok
	case *v1.Service:
		m := Meta{name: obj.Name, namespace: obj.Namespace}
		_, ok := kc.services[m]
		delete(kc.services, m)
		return ok
	case *v1beta1.Ingress:
		m := Meta{name: obj.Name, namespace: obj.Namespace}
		_, ok := kc.ingresses[m]
		delete(kc.ingresses, m)
		return ok
	case *ingressroutev1.IngressRoute:
		m := Meta{name: obj.Name, namespace: obj.Namespace}
		_, ok := kc.ingressroutes[m]
		delete(kc.ingressroutes, m)
		return ok
	case *ingressroutev1.TLSCertificateDelegation:
		m := Meta{name: obj.Name, namespace: obj.Namespace}
		_, ok := kc.delegations[m]
		delete(kc.delegations, m)
		return ok
	default:
		// not interesting
		return false
	}
}
