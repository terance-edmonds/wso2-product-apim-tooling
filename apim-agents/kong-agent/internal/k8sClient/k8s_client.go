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

// Package k8sclient contains the common implementation methods to invoke k8s APIs in the agent
package k8sclient

import (
	"context"

	v1 "github.com/kong/kubernetes-configuration/api/configuration/v1"
	"github.com/wso2/product-apim-tooling/apim-agent/config"
	"github.com/wso2/product-apim-tooling/apim-agents/kong-agent/internal/loggers"
	"github.com/wso2/product-apim-tooling/apim-agents/kong-agent/pkg/utils"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	k8error "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// DeployHTTPRouteCR applies the given HttpRoute struct to the Kubernetes cluster.
func DeployHTTPRouteCR(httpRoute *gwapiv1.HTTPRoute, k8sClient client.Client) {
	crHTTPRoute := &gwapiv1.HTTPRoute{}
	yamlCR, err := yaml.Marshal(httpRoute)
	if err != nil {
		loggers.LoggerK8sClient.Error(err)
	}
	loggers.LoggerK8sClient.Infof("========== HTTPRoute\n %v \n", string(yamlCR))
	if err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: httpRoute.ObjectMeta.Namespace, Name: httpRoute.Name}, crHTTPRoute); err != nil {
		if !k8error.IsNotFound(err) {
			loggers.LoggerK8sClient.Error("Unable to get HTTPRoute CR: " + err.Error())
		}
		if err := k8sClient.Create(context.Background(), httpRoute); err != nil {
			loggers.LoggerK8sClient.Error("Unable to create HTTPRoute CR: " + err.Error())
		} else {
			loggers.LoggerK8sClient.Info("HTTPRoute CR created: " + httpRoute.Name)
		}
	} else {
		crHTTPRoute.Spec = httpRoute.Spec
		if err := k8sClient.Update(context.Background(), crHTTPRoute); err != nil {
			loggers.LoggerK8sClient.Error("Unable to update HTTPRoute CR: " + err.Error())
		} else {
			loggers.LoggerK8sClient.Info("HTTPRoute CR updated: " + crHTTPRoute.Name)
		}
	}
}

// DeployServiceCR applies the given Service struct to the Kubernetes cluster.
func DeployServiceCR(service *corev1.Service, k8sClient client.Client) {
	yamlCR, err := yaml.Marshal(service)
	if err != nil {
		loggers.LoggerK8sClient.Error(err)
	}
	loggers.LoggerK8sClient.Infof("========== Service\n %v \n", string(yamlCR))
	crService := &corev1.Service{}
	if err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: service.ObjectMeta.Namespace, Name: service.Name}, crService); err != nil {
		if !k8error.IsNotFound(err) {
			loggers.LoggerK8sClient.Error("Unable to get Service CR: " + err.Error())
		}
		if err := k8sClient.Create(context.Background(), service); err != nil {
			loggers.LoggerK8sClient.Error("Unable to create Service CR: " + err.Error())
		} else {
			loggers.LoggerK8sClient.Info("Service CR created: " + service.Name)
		}
	} else {
		crService.Spec = service.Spec
		if err := k8sClient.Update(context.Background(), crService); err != nil {
			loggers.LoggerK8sClient.Error("Unable to update Service CR: " + err.Error())
		} else {
			loggers.LoggerK8sClient.Info("Service CR updated: " + crService.Name)
		}
	}
}

// DeployKongPluginCR applies the given KongPlugin struct to the Kubernetes cluster.
func DeployKongPluginCR(plugin *v1.KongPlugin, k8sClient client.Client) {
	yamlCR, err := yaml.Marshal(plugin)
	if err != nil {
		loggers.LoggerK8sClient.Error(err)
	}
	loggers.LoggerK8sClient.Infof("========== Plugin\n %v \n", string(yamlCR))
	crKongPlugin := &v1.KongPlugin{}
	if err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: plugin.ObjectMeta.Namespace, Name: plugin.Name}, crKongPlugin); err != nil {
		if !k8error.IsNotFound(err) {
			loggers.LoggerK8sClient.Error("Unable to get KongPlugin CR: " + err.Error())
		}
		if err := k8sClient.Create(context.Background(), plugin); err != nil {
			loggers.LoggerK8sClient.Error("Unable to create KongPlugin CR: " + err.Error())
		} else {
			loggers.LoggerK8sClient.Info("KongPlugin CR created: " + plugin.Name)
		}
	} else {
		crKongPlugin.Config = plugin.Config
		if err := k8sClient.Update(context.Background(), crKongPlugin); err != nil {
			loggers.LoggerK8sClient.Error("Unable to update KongPlugin CR: " + err.Error())
		} else {
			loggers.LoggerK8sClient.Info("KongPlugin CR updated: " + crKongPlugin.Name)
		}
	}
}

// DeployKongConsumerCR applies the given KongConsumer struct to the Kubernetes cluster.
func DeployKongConsumerCR(consumer *v1.KongConsumer, k8sClient client.Client) {
	yamlCR, err := yaml.Marshal(consumer)
	if err != nil {
		loggers.LoggerK8sClient.Error(err)
	}
	loggers.LoggerK8sClient.Infof("========== Consumer\n %v \n", string(yamlCR))
	crKongConsumer := &v1.KongConsumer{}
	if err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: consumer.ObjectMeta.Namespace, Name: consumer.Name}, crKongConsumer); err != nil {
		if !k8error.IsNotFound(err) {
			loggers.LoggerK8sClient.Error("Unable to get KongConsumer CR: " + err.Error())
		}
		if err := k8sClient.Create(context.Background(), consumer); err != nil {
			loggers.LoggerK8sClient.Error("Unable to create KongConsumer CR: " + err.Error())
		} else {
			loggers.LoggerK8sClient.Info("KongConsumer CR created: " + consumer.Name)
		}
	} else {
		crKongConsumer.CustomID = consumer.CustomID
		crKongConsumer.Username = consumer.Username
		if err := k8sClient.Update(context.Background(), crKongConsumer); err != nil {
			loggers.LoggerK8sClient.Error("Unable to update KongConsumer CR: " + err.Error())
		} else {
			loggers.LoggerK8sClient.Info("KongConsumer CR updated: " + crKongConsumer.Name)
		}
	}
}

// UndeployAPICRs removes the API Custom Resources from the Kubernetes cluster based on API ID label.
func UndeployAPICRs(apiID string, k8sClient client.Client) {
	conf, errReadConfig := config.ReadConfigs()
	if errReadConfig != nil {
		loggers.LoggerK8sClient.Errorf("Error reading configurations: %v", errReadConfig)
	}

	undeployHTTPRoutes(apiID, k8sClient, conf)
	undeployServices(apiID, k8sClient, conf)
	undeployKongPlugins(k8sClient, conf, labels.SelectorFromSet(map[string]string{"apiUUID": apiID}))
}

// UndeployAPPCRs removes the APP Custom Resources from the Kubernetes cluster based on Application ID label.
func UndeployAPPCRs(appID string, k8sClient client.Client) {
	conf, errReadConfig := config.ReadConfigs()
	if errReadConfig != nil {
		loggers.LoggerK8sClient.Errorf("Error reading configurations: %v", errReadConfig)
	}
	undeployKongConsumers(appID, k8sClient, conf)
	undeployKongPlugins(k8sClient, conf, labels.SelectorFromSet(map[string]string{"applicationUUID": appID}))
}

// undeployHTTPRoutes removes the HTTPRoute Resources from the Kubernetes cluster based on API ID label.
func undeployHTTPRoutes(apiID string, k8sClient client.Client, conf *config.Config) {
	resourceList := &gwapiv1.HTTPRouteList{}
	err := k8sClient.List(context.Background(), resourceList, &client.ListOptions{Namespace: conf.DataPlane.Namespace, LabelSelector: labels.SelectorFromSet(map[string]string{"apiUUID": apiID})})
	// Retrieve all CRs from the Kubernetes cluster
	if err != nil {
		loggers.LoggerK8sClient.Errorf("Unable to list HTTPRoute CRs: %v", err)
	} else {
		for _, resource := range resourceList.Items {
			err := k8sClient.Delete(context.Background(), &resource, &client.DeleteOptions{})
			if err != nil {
				loggers.LoggerK8sClient.Errorf("Unable to delete HTTPRoute CR: %v", err)
			} else {
				loggers.LoggerK8sClient.Infof("Deleted HTTPRoute CR: %s", resource.Name)
			}
		}
	}
}

// undeployServices removes the Service Resources from the Kubernetes cluster based on API ID label.
func undeployServices(apiID string, k8sClient client.Client, conf *config.Config) {
	resourceList := &corev1.ServiceList{}
	err := k8sClient.List(context.Background(), resourceList, &client.ListOptions{Namespace: conf.DataPlane.Namespace, LabelSelector: labels.SelectorFromSet(map[string]string{"apiUUID": apiID})})
	// Retrieve all CRs from the Kubernetes cluster
	if err != nil {
		loggers.LoggerK8sClient.Errorf("Unable to list Service CRs: %v", err)
	} else {
		for _, resource := range resourceList.Items {
			err := k8sClient.Delete(context.Background(), &resource, &client.DeleteOptions{})
			if err != nil {
				loggers.LoggerK8sClient.Errorf("Unable to delete Service CR: %v", err)
			} else {
				loggers.LoggerK8sClient.Infof("Deleted Service CR: %s", resource.Name)
			}
		}
	}
}

// undeployKongPlugins removes the KongPlugin Resources from the Kubernetes cluster based on label selector.
func undeployKongPlugins(k8sClient client.Client, conf *config.Config, labelSelector labels.Selector) {
	resourceList := &v1.KongPluginList{}
	err := k8sClient.List(context.Background(), resourceList, &client.ListOptions{Namespace: conf.DataPlane.Namespace, LabelSelector: labelSelector})
	// Retrieve all CRs from the Kubernetes cluster
	if err != nil {
		loggers.LoggerK8sClient.Errorf("Unable to list KongPlugin CRs: %v", err)
	} else {
		for _, resource := range resourceList.Items {
			err := k8sClient.Delete(context.Background(), &resource, &client.DeleteOptions{})
			if err != nil {
				loggers.LoggerK8sClient.Errorf("Unable to delete KongPlugin CR: %v", err)
			} else {
				loggers.LoggerK8sClient.Infof("Deleted KongPlugin CR: %s", resource.Name)
			}
		}
	}
}

// undeployKongConsumers removes the KongConsumer Resources from the Kubernetes cluster based on application ID label.
func undeployKongConsumers(appID string, k8sClient client.Client, conf *config.Config) {
	resourceList := &v1.KongConsumerList{}
	err := k8sClient.List(context.Background(), resourceList, &client.ListOptions{Namespace: conf.DataPlane.Namespace, LabelSelector: labels.SelectorFromSet(map[string]string{"applicationUUID": appID})})
	// Retrieve all CRs from the Kubernetes cluster
	if err != nil {
		loggers.LoggerK8sClient.Errorf("Unable to list KongConsumer CRs: %v", err)
	} else {
		for _, resource := range resourceList.Items {
			err := k8sClient.Delete(context.Background(), &resource, &client.DeleteOptions{})
			if err != nil {
				loggers.LoggerK8sClient.Errorf("Unable to delete KongConsumer CR: %v", err)
			} else {
				loggers.LoggerK8sClient.Infof("Deleted KongConsumer CR: %s", resource.Name)
			}
		}
	}
}

// UpdateKongConsumerCredential adds new credentials to KongConsumer Resources from the Kubernetes cluster based on application ID label.
func UpdateKongConsumerCredential(appID string, k8sClient client.Client, conf *config.Config, credentials []string) {
	resourceList := &v1.KongConsumerList{}
	err := k8sClient.List(context.Background(), resourceList, &client.ListOptions{Namespace: conf.DataPlane.Namespace, LabelSelector: labels.SelectorFromSet(map[string]string{"applicationUUID": appID})})
	// Retrieve all CRs from the Kubernetes cluster
	if err != nil {
		loggers.LoggerK8sClient.Errorf("Unable to list KongConsumer CRs: %v", err)
	} else {
		for _, resource := range resourceList.Items {
			// update credentials
			resource.Credentials = append(resource.Credentials, credentials...)
			err := k8sClient.Update(context.Background(), &resource, &client.UpdateOptions{})
			if err != nil {
				loggers.LoggerK8sClient.Errorf("Unable to delete KongConsumer CR: %v", err)
			} else {
				loggers.LoggerK8sClient.Infof("Updated KongConsumer CR: %s", resource.Name)
			}
		}
	}
}

// RemoveKongConsumerCredential removes credentials from KongConsumer Resources from the Kubernetes cluster based on application ID label.
func RemoveKongConsumerCredential(appID string, k8sClient client.Client, conf *config.Config, credentials []string) {
	resourceList := &v1.KongConsumerList{}
	err := k8sClient.List(context.Background(), resourceList, &client.ListOptions{Namespace: conf.DataPlane.Namespace, LabelSelector: labels.SelectorFromSet(map[string]string{"applicationUUID": appID})})
	// Retrieve all CRs from the Kubernetes cluster
	if err != nil {
		loggers.LoggerK8sClient.Errorf("Unable to list KongConsumer CRs: %v", err)
	} else {
		for _, resource := range resourceList.Items {
			// update credentials
			resource.Credentials = utils.FilterItems(resource.Credentials, credentials)
			err := k8sClient.Update(context.Background(), &resource, &client.UpdateOptions{})
			if err != nil {
				loggers.LoggerK8sClient.Errorf("Unable to delete KongConsumer CR: %v", err)
			} else {
				loggers.LoggerK8sClient.Infof("Updated KongConsumer CR: %s", resource.Name)
			}
		}
	}
}
