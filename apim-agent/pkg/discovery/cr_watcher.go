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

package discovery

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/wso2/product-apim-tooling/apim-agent/config"
	"github.com/wso2/product-apim-tooling/apim-agent/pkg/loggers"
	"github.com/wso2/product-apim-tooling/apim-agent/pkg/tlsutils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// Define the resources to watch
var (
	configOnce   sync.Once
	eventQueue   chan APICPEvent
	APIMap       map[string]API // Maps apiUUID to latest API struct
	host         string
	port         uint16
	apisRestPath string
	skipSSL      bool
	wg           sync.WaitGroup
)

// CRWatcher defines a watcher for Kubernetes Custom Resources with pluggable event handlers
type CRWatcher struct {
	DynamicClient dynamic.Interface
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
	cw.DynamicClient = dynamicClient

	ctx := context.Background()

	// Start watching each resource
	for _, gvr := range cw.GroupVersions {
		go cw.watchResource(ctx, cw.DynamicClient, gvr, cw.Namespace)
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

func init() {
	configOnce.Do(func() {
		conf, _ := config.ReadConfigs()
		APIMap = make(map[string]API)
		eventQueue = make(chan APICPEvent, 100)
		host = conf.ControlPlane.Host
		port = conf.ControlPlane.RestPort
		skipSSL = conf.ControlPlane.SkipSSLVerification
		apisRestPath = fmt.Sprintf("https://%s:%d%s", host, port, conf.ControlPlane.APIsRestPath)

		wg.Add(1)
		go sendData()
	})
}

// sendData sends data as a POST request to the control plane host.
func sendData() {
	loggers.LoggerWatcher.Infof("A thread assigned to send API events to agent")
	tr := &http.Transport{}
	if !skipSSL {
		_, _, truststoreLocation := tlsutils.GetKeyLocations()
		caCertPool := tlsutils.GetTrustedCertPool(truststoreLocation)
		tr = &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: caCertPool},
		}
	} else {
		tr = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	// Configuring the http client
	client := &http.Client{
		Transport: tr,
	}
	defer wg.Done()

	for event := range eventQueue {
		loggers.LoggerWatcher.Infof("Sending api event to agent. Event: %+v", event)
		jsonData, err := json.Marshal(event)
		if err != nil {
			loggers.LoggerWatcher.Errorf("Error marshalling data. Error %+v", err)
			continue
		}
		for {
			resp, err := client.Post(
				apisRestPath,
				applicationJSON,
				bytes.NewBuffer(jsonData),
			)
			if err != nil {
				loggers.LoggerWatcher.Errorf("Error sending data. Error: %+v, Retrying after %d seconds", err, retryInterval)
				time.Sleep(time.Second * retryInterval)
				continue
			}
			defer resp.Body.Close()
			body, _ := ioutil.ReadAll(resp.Body)
			if resp.StatusCode == http.StatusServiceUnavailable {
				loggers.LoggerWatcher.Errorf("Error: Unexpected status code: %d, received message: %s, retrying after %d seconds", resp.StatusCode, string(body), retryInterval)
				time.Sleep(time.Second * retryInterval)
				continue
			}
			if event.Event == EventTypeDelete {
				// If it’s a delete event that got propagated to CP, no further action needed
				break
			}
			// Removed delete(apiHashMap, event.API.APIHash) since apiHashMap is no longer used

			var responseMap map[string]interface{}
			if err := json.Unmarshal([]byte(body), &responseMap); err != nil {
				loggers.LoggerWatcher.Errorf("Could not decode response body as json. body: %+v", string(body))
				break
			}
			// Extract id and revisionID from response
			id, ok := responseMap["id"].(string)
			revisionID, revisionOk := responseMap["revisionID"].(string)
			if !ok {
				loggers.LoggerWatcher.Errorf("Id field not present in response body. encoded body: %+v", responseMap)
				id = "" // Default to empty string if not found
			}
			if !revisionOk {
				loggers.LoggerWatcher.Errorf("Revision field not present in response body. encoded body: %+v", responseMap)
				revisionID = ""
			}
			loggers.LoggerWatcher.Infof("Adding label update to API %s/%s, Labels: apiUUID: %s, revisionID: %s, apiHash: %s",
				event.CRNamespace, event.CRName, id, revisionID, event.API.APIHash)
			break
		}
	}
}

// QueueEvent adds an event to the event queue
func QueueEvent(eventType EventType, api API, crName, crNamespace string) {
	event := APICPEvent{
		Event:       eventType,
		API:         api,
		CRName:      crName,
		CRNamespace: crNamespace,
	}
	select {
	case eventQueue <- event:
		loggers.LoggerWatcher.Infof("Queued %s event for API %s", eventType, api.APIUUID)
	default:
		loggers.LoggerWatcher.Warnf("Event queue full, dropping %s event for API %s", eventType, api.APIUUID)
	}
}
