/*
 *  Copyright (c) 2024, WSO2 LLC. (http://www.wso2.org) All Rights Reserved.
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *  http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 *
 */

package watcher

import (
	"context"
	"time"

	"github.com/wso2/product-apim-tooling/apim-agent/pkg/loggers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// CRWatcher defines a watcher for Kubernetes Custom Resources with pluggable event handlers
type CRWatcher struct {
	Namespace     string
	GroupVersions []schema.GroupVersionResource
	AddFunc       func(*unstructured.Unstructured)
	UpdateFunc    func(oldObj, newObj *unstructured.Unstructured)
	DeleteFunc    func(*unstructured.Unstructured)
}

// Watch starts watching the specified resources with the provided handlers
func (cw *CRWatcher) Watch() {
	// Load in-cluster Kubernetes config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err)
	}

	// Create dynamic Kubernetes client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	// Start watching each resource
	for _, gvr := range cw.GroupVersions {
		go cw.watchResource(ctx, dynamicClient, gvr, cw.Namespace)
	}

	// Keep running
	select {}
}

// watchResource watches a specific GVR in a namespace
func (cw *CRWatcher) watchResource(ctx context.Context, client dynamic.Interface, gvr schema.GroupVersionResource, namespace string) {
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return client.Resource(gvr).Namespace(namespace).List(ctx, options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return client.Resource(gvr).Namespace(namespace).Watch(ctx, options)
			},
		},
		&unstructured.Unstructured{},
		time.Minute, // Resync period
		cache.Indexers{},
	)

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			u := obj.(*unstructured.Unstructured)
			if cw.AddFunc != nil {
				cw.AddFunc(u)
			} else {
				loggers.LoggerWatcher.Printf("%s Added: %s/%s\n", gvr.Resource, u.GetNamespace(), u.GetName())
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldU := oldObj.(*unstructured.Unstructured)
			newU := newObj.(*unstructured.Unstructured)
			if cw.UpdateFunc != nil {
				cw.UpdateFunc(oldU, newU)
			} else {
				loggers.LoggerWatcher.Printf("%s Updated: %s/%s\n", gvr.Resource, newU.GetNamespace(), newU.GetName())
			}
		},
		DeleteFunc: func(obj interface{}) {
			u := obj.(*unstructured.Unstructured)
			if cw.DeleteFunc != nil {
				cw.DeleteFunc(u)
			} else {
				loggers.LoggerWatcher.Printf("%s Deleted: %s/%s\n", gvr.Resource, u.GetNamespace(), u.GetName())
			}
		},
	})

	informer.Run(ctx.Done())
}
