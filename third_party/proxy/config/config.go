/*
Copyright 2014 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
/*
// Copyright 2020 Antrea Authors
//
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

Modifies:
- Replace "k8s.io/kubernetes/pkg/controller" to "k8s.io/client-go/tools/cache"
*/

package config

import (
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	discovery "k8s.io/api/discovery/v1beta1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	coreinformers "k8s.io/client-go/informers/core/v1"
	discoveryinformers "k8s.io/client-go/informers/discovery/v1beta1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

// ServiceHandler is an abstract interface of objects which receive
// notifications about service object changes.
type ServiceHandler interface {
	// OnServiceAdd is called whenever creation of new service object
	// is observed.
	OnServiceAdd(service *v1.Service)
	// OnServiceUpdate is called whenever modification of an existing
	// service object is observed.
	OnServiceUpdate(oldService, service *v1.Service)
	// OnServiceDelete is called whenever deletion of an existing service
	// object is observed.
	OnServiceDelete(service *v1.Service)
	// OnServiceSynced is called once all the initial event handlers were
	// called and the state is fully propagated to local cache.
	OnServiceSynced()
}

// EndpointsHandler is an abstract interface of objects which receive
// notifications about endpoints object changes.
type EndpointsHandler interface {
	// OnEndpointsAdd is called whenever creation of new endpoints object
	// is observed.
	OnEndpointsAdd(endpoints *v1.Endpoints)
	// OnEndpointsUpdate is called whenever modification of an existing
	// endpoints object is observed.
	OnEndpointsUpdate(oldEndpoints, endpoints *v1.Endpoints)
	// OnEndpointsDelete is called whenever deletion of an existing endpoints
	// object is observed.
	OnEndpointsDelete(endpoints *v1.Endpoints)
	// OnEndpointsSynced is called once all the initial event handlers were
	// called and the state is fully propagated to local cache.
	OnEndpointsSynced()
}

// EndpointSliceHandler is an abstract interface of objects which receive
// notifications about endpoint slice object changes.
type EndpointSliceHandler interface {
	// OnEndpointSliceAdd is called whenever creation of new endpoint slice
	// object is observed.
	OnEndpointSliceAdd(endpointSlice *discovery.EndpointSlice)
	// OnEndpointSliceUpdate is called whenever modification of an existing
	// endpoint slice object is observed.
	OnEndpointSliceUpdate(oldEndpointSlice, newEndpointSlice *discovery.EndpointSlice)
	// OnEndpointSliceDelete is called whenever deletion of an existing
	// endpoint slice object is observed.
	OnEndpointSliceDelete(endpointSlice *discovery.EndpointSlice)
	// OnEndpointSlicesSynced is called once all the initial event handlers were
	// called and the state is fully propagated to local cache.
	OnEndpointSlicesSynced()
}

// EndpointsConfig tracks a set of endpoints configurations.
type EndpointsConfig struct {
	listerSynced  cache.InformerSynced
	eventHandlers []EndpointsHandler
}

// NewEndpointsConfig creates a new EndpointsConfig.
func NewEndpointsConfig(endpointsInformer coreinformers.EndpointsInformer, resyncPeriod time.Duration) *EndpointsConfig {
	result := &EndpointsConfig{
		listerSynced: endpointsInformer.Informer().HasSynced,
	}

	endpointsInformer.Informer().AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    result.handleAddEndpoints,
			UpdateFunc: result.handleUpdateEndpoints,
			DeleteFunc: result.handleDeleteEndpoints,
		},
		resyncPeriod,
	)

	return result
}

// RegisterEventHandler registers a handler which is called on every endpoints change.
func (c *EndpointsConfig) RegisterEventHandler(handler EndpointsHandler) {
	c.eventHandlers = append(c.eventHandlers, handler)
}

// Run waits for cache synced and invokes handlers after syncing.
func (c *EndpointsConfig) Run(stopCh <-chan struct{}) {
	klog.Info("Starting endpoints config controller")

	if !cache.WaitForNamedCacheSync("endpoints config", stopCh, c.listerSynced) {
		return
	}

	for i := range c.eventHandlers {
		klog.V(3).Infof("Calling handler.OnEndpointsSynced()")
		c.eventHandlers[i].OnEndpointsSynced()
	}
}

func (c *EndpointsConfig) handleAddEndpoints(obj interface{}) {
	endpoints, ok := obj.(*v1.Endpoints)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
		return
	}
	for i := range c.eventHandlers {
		klog.V(4).Infof("Calling handler.OnEndpointsAdd")
		c.eventHandlers[i].OnEndpointsAdd(endpoints)
	}
}

func (c *EndpointsConfig) handleUpdateEndpoints(oldObj, newObj interface{}) {
	oldEndpoints, ok := oldObj.(*v1.Endpoints)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", oldObj))
		return
	}
	endpoints, ok := newObj.(*v1.Endpoints)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", newObj))
		return
	}
	for i := range c.eventHandlers {
		klog.V(4).Infof("Calling handler.OnEndpointsUpdate")
		c.eventHandlers[i].OnEndpointsUpdate(oldEndpoints, endpoints)
	}
}

func (c *EndpointsConfig) handleDeleteEndpoints(obj interface{}) {
	endpoints, ok := obj.(*v1.Endpoints)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
			return
		}
		if endpoints, ok = tombstone.Obj.(*v1.Endpoints); !ok {
			utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
			return
		}
	}
	for i := range c.eventHandlers {
		klog.V(4).Infof("Calling handler.OnEndpointsDelete")
		c.eventHandlers[i].OnEndpointsDelete(endpoints)
	}
}

// ServiceConfig tracks a set of service configurations.
type ServiceConfig struct {
	listerSynced  cache.InformerSynced
	eventHandlers []ServiceHandler
}

// NewServiceConfig creates a new ServiceConfig.
func NewServiceConfig(serviceInformer coreinformers.ServiceInformer, resyncPeriod time.Duration) *ServiceConfig {
	result := &ServiceConfig{
		listerSynced: serviceInformer.Informer().HasSynced,
	}

	serviceInformer.Informer().AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    result.handleAddService,
			UpdateFunc: result.handleUpdateService,
			DeleteFunc: result.handleDeleteService,
		},
		resyncPeriod,
	)

	return result
}

// RegisterEventHandler registers a handler which is called on every service change.
func (c *ServiceConfig) RegisterEventHandler(handler ServiceHandler) {
	c.eventHandlers = append(c.eventHandlers, handler)
}

// Run waits for cache synced and invokes handlers after syncing.
func (c *ServiceConfig) Run(stopCh <-chan struct{}) {
	klog.Info("Starting service config controller")

	if !cache.WaitForNamedCacheSync("service config", stopCh, c.listerSynced) {
		return
	}

	for i := range c.eventHandlers {
		klog.V(3).Info("Calling handler.OnServiceSynced()")
		c.eventHandlers[i].OnServiceSynced()
	}
}

func (c *ServiceConfig) handleAddService(obj interface{}) {
	service, ok := obj.(*v1.Service)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
		return
	}
	for i := range c.eventHandlers {
		klog.V(4).Info("Calling handler.OnServiceAdd")
		c.eventHandlers[i].OnServiceAdd(service)
	}
}

func (c *ServiceConfig) handleUpdateService(oldObj, newObj interface{}) {
	oldService, ok := oldObj.(*v1.Service)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", oldObj))
		return
	}
	service, ok := newObj.(*v1.Service)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", newObj))
		return
	}
	for i := range c.eventHandlers {
		klog.V(4).Info("Calling handler.OnServiceUpdate")
		c.eventHandlers[i].OnServiceUpdate(oldService, service)
	}
}

func (c *ServiceConfig) handleDeleteService(obj interface{}) {
	service, ok := obj.(*v1.Service)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
			return
		}
		if service, ok = tombstone.Obj.(*v1.Service); !ok {
			utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
			return
		}
	}
	for i := range c.eventHandlers {
		klog.V(4).Info("Calling handler.OnServiceDelete")
		c.eventHandlers[i].OnServiceDelete(service)
	}
}

// EndpointSliceConfig tracks a set of endpoints configurations.
type EndpointSliceConfig struct {
	listerSynced  cache.InformerSynced
	eventHandlers []EndpointSliceHandler
}

// NewEndpointSliceConfig creates a new EndpointSliceConfig.
func NewEndpointSliceConfig(endpointSliceInformer discoveryinformers.EndpointSliceInformer, resyncPeriod time.Duration) *EndpointSliceConfig {
	result := &EndpointSliceConfig{
		listerSynced: endpointSliceInformer.Informer().HasSynced,
	}

	endpointSliceInformer.Informer().AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    result.handleAddEndpointSlice,
			UpdateFunc: result.handleUpdateEndpointSlice,
			DeleteFunc: result.handleDeleteEndpointSlice,
		},
		resyncPeriod,
	)

	return result
}

// RegisterEventHandler registers a handler which is called on every endpoint slice change.
func (c *EndpointSliceConfig) RegisterEventHandler(handler EndpointSliceHandler) {
	c.eventHandlers = append(c.eventHandlers, handler)
}

// Run waits for cache synced and invokes handlers after syncing.
func (c *EndpointSliceConfig) Run(stopCh <-chan struct{}) {
	klog.Info("Starting endpoint slice config controller")

	if !cache.WaitForNamedCacheSync("endpoint slice config", stopCh, c.listerSynced) {
		return
	}

	for _, h := range c.eventHandlers {
		klog.V(3).Infof("Calling handler.OnEndpointSlicesSynced()")
		h.OnEndpointSlicesSynced()
	}
}

func (c *EndpointSliceConfig) handleAddEndpointSlice(obj interface{}) {
	endpointSlice, ok := obj.(*discovery.EndpointSlice)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %T", obj))
		return
	}
	for _, h := range c.eventHandlers {
		klog.V(4).InfoS("Calling handler.OnEndpointSliceAdd", "endpointSlice", endpointSlice)
		h.OnEndpointSliceAdd(endpointSlice)
	}
}

func (c *EndpointSliceConfig) handleUpdateEndpointSlice(oldObj, newObj interface{}) {
	oldEndpointSlice, ok := oldObj.(*discovery.EndpointSlice)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %T", newObj))
		return
	}
	newEndpointSlice, ok := newObj.(*discovery.EndpointSlice)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %T", newObj))
		return
	}
	for _, h := range c.eventHandlers {
		klog.V(4).InfoS("Calling handler.OnEndpointSliceUpdate", "from", oldEndpointSlice, "to", newEndpointSlice)
		h.OnEndpointSliceUpdate(oldEndpointSlice, newEndpointSlice)
	}
}

func (c *EndpointSliceConfig) handleDeleteEndpointSlice(obj interface{}) {
	endpointSlice, ok := obj.(*discovery.EndpointSlice)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("unexpected object type: %T", obj))
			return
		}
		if endpointSlice, ok = tombstone.Obj.(*discovery.EndpointSlice); !ok {
			utilruntime.HandleError(fmt.Errorf("unexpected object type: %T", obj))
			return
		}
	}
	for _, h := range c.eventHandlers {
		klog.V(4).InfoS("Calling handler.OnEndpointSliceDelete", "endpointSlice", endpointSlice)
		h.OnEndpointSliceDelete(endpointSlice)
	}
}
