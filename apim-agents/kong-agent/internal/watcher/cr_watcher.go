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
	"github.com/wso2/product-apim-tooling/apim-agent/config"
	"github.com/wso2/product-apim-tooling/apim-agent/pkg/loggers"
	watcherPkg "github.com/wso2/product-apim-tooling/apim-agent/pkg/watcher"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Define the resources to watch
var gvrs = []schema.GroupVersionResource{
	{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "httproutes"},
	{Group: "configuration.konghq.com", Version: "v1", Resource: "kongplugins"},
	{Group: "configuration.konghq.com", Version: "v1", Resource: "kongconsumers"},
}

// addResource handles the addition of a resource
func addResource(u *unstructured.Unstructured) {
	loggers.LoggerWatcher.Infof("Resource Added: %s/%s (Kind: %s)\n", u.GetNamespace(), u.GetName(), u.GetKind())
	if u.GetKind() == "KongPlugin" {
		if plugin, found, _ := unstructured.NestedString(u.Object, "plugin"); found {
			loggers.LoggerWatcher.Infof("  Plugin Type: %s\n", plugin)
		}
	}
}

// updateResource handles the update of a resource
func updateResource(oldU, newU *unstructured.Unstructured) {
	loggers.LoggerWatcher.Infof("Resource Updated: %s/%s (Kind: %s)\n", newU.GetNamespace(), newU.GetName(), newU.GetKind())
	// Example: Check for changes in KongPlugin configuration
	if newU.GetKind() == "KongPlugin" {
		oldPlugin, _, _ := unstructured.NestedString(oldU.Object, "plugin")
		newPlugin, _, _ := unstructured.NestedString(newU.Object, "plugin")
		if oldPlugin != newPlugin {
			loggers.LoggerWatcher.Infof("  Plugin Type Changed: %s -> %s\n", oldPlugin, newPlugin)
		}
	}
}

// deleteResource handles the deletion of a resource
func deleteResource(u *unstructured.Unstructured) {
	loggers.LoggerWatcher.Infof("Resource Deleted: %s/%s (Kind: %s)\n", u.GetNamespace(), u.GetName(), u.GetKind())
}

// Initialize CRWatcher with separate handler functions
var CRWatcher *watcherPkg.CRWatcher

func init() {
	conf, _ := config.ReadConfigs()

	CRWatcher = &watcherPkg.CRWatcher{
		Namespace:     conf.DataPlane.Namespace,
		GroupVersions: gvrs,
		AddFunc:       addResource,
		UpdateFunc:    updateResource,
		DeleteFunc:    deleteResource,
	}
}
