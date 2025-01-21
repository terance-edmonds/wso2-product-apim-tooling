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

// Package agent contains the implementation to start the agent
package agent

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"sync"
	"time"

	cpv1alpha2 "github.com/wso2/apk/common-go-libs/apis/cp/v1alpha2"
	dpv1alpha1 "github.com/wso2/apk/common-go-libs/apis/dp/v1alpha1"
	dpv1alpha2 "github.com/wso2/apk/common-go-libs/apis/dp/v1alpha2"
	dpv1alpha3 "github.com/wso2/apk/common-go-libs/apis/dp/v1alpha3"
	"github.com/wso2/apk/common-go-libs/pkg/discovery/api/wso2/discovery/service/apkmgt"
	"github.com/wso2/product-apim-tooling/apim-agent/config"
	"github.com/wso2/product-apim-tooling/apim-apk-agent/internal/eventhub"
	logger "github.com/wso2/product-apim-tooling/apim-apk-agent/internal/loggers"
	logging "github.com/wso2/product-apim-tooling/apim-apk-agent/internal/logging"
	"github.com/wso2/product-apim-tooling/apim-apk-agent/internal/synchronizer"
	"github.com/wso2/product-apim-tooling/apim-apk-agent/pkg/managementserver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	debug       bool
	onlyLogging bool

	port     uint
	alsPort  uint
	restPort uint

	mode string
)

const (
	ads                      = "ads"
	amqpProtocol             = "amqp"
	grpcMaxConcurrentStreams = 1000000
)

func init() {
	flag.BoolVar(&debug, "debug", true, "Use debug logging")
	flag.BoolVar(&onlyLogging, "onlyLogging", false, "Only demo AccessLogging Service")
	flag.UintVar(&port, "port", 18000, "Management server port")
	flag.UintVar(&alsPort, "als", 18090, "Accesslog server port")
	flag.StringVar(&mode, "ads", ads, "Management server type (ads, grpc, rest)")
	flag.UintVar(&restPort, "rest_port", 18001, "Rest server port")
}

// PreRun prepares the agent environment and runs before Run.
func PreRun(conf *config.Config, scheme *runtime.Scheme) {
	utilruntime.Must(dpv1alpha1.AddToScheme(scheme))
	utilruntime.Must(dpv1alpha2.AddToScheme(scheme))
	utilruntime.Must(dpv1alpha3.AddToScheme(scheme))
	utilruntime.Must(cpv1alpha2.AddToScheme(scheme))
	utilruntime.Must(cpv1alpha2.AddToScheme(scheme))
	utilruntime.Must(dpv1alpha3.AddToScheme(scheme))
}

// Run starts the GRPC server and Rest API server.
func Run(conf *config.Config, mgr manager.Manager) {
	// Start the manager in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.LoggerAgent.Info("starting manager")
		if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
			logger.LoggerAgent.Warnf("problem running manager: %v", err)
		}
	}()

	AgentMode := conf.Agent.Mode
	logger.LoggerAgent.Infof("Agent Mode: %v", AgentMode)

	if AgentMode == "CPtoDP" {
		// Load initial Policy data from control plane
		synchronizer.FetchRateLimitPoliciesOnEvent("", "", mgr.GetClient())
	}
	// Load initial Subscription Rate Limit data from control plane
	synchronizer.FetchSubscriptionRateLimitPoliciesOnEvent("", "", mgr.GetClient(), true)
	// Load initial AI Provider data from control plane
	synchronizer.FetchAIProvidersOnEvent("", "", "", mgr.GetClient(), true)

	// Load initial data from control plane
	eventhub.LoadInitialData(conf, mgr.GetClient())

	// Load initial KM data from control plane
	synchronizer.FetchKeyManagersOnStartUp(mgr.GetClient())

	var grpcOptions []grpc.ServerOption
	grpcOptions = append(grpcOptions, grpc.KeepaliveParams(
		keepalive.ServerParameters{
			Time:    time.Duration(5 * time.Minute),
			Timeout: time.Duration(20 * time.Second),
		}),
		grpc.MaxConcurrentStreams(grpcMaxConcurrentStreams),
	)
	publicKeyLocation, privateKeyLocation, truststoreLocation := config.GetKeyLocations()
	cert, err := config.GetServerCertificate(publicKeyLocation, privateKeyLocation)

	caCertPool := config.GetTrustedCertPool(truststoreLocation)

	if err == nil {
		grpcOptions = append(grpcOptions, grpc.Creds(
			credentials.NewTLS(&tls.Config{
				Certificates: []tls.Certificate{cert},
				ClientAuth:   tls.RequireAndVerifyClientCert,
				ClientCAs:    caCertPool,
			}),
		))
	} else {
		logger.LoggerAgent.Warn("failed to initiate the ssl context: ", err)
		panic(err)
	}

	grpcOptions = append(grpcOptions, grpc.KeepaliveParams(
		keepalive.ServerParameters{
			Time:    time.Duration(5 * time.Minute),
			Timeout: time.Duration(20 * time.Second),
		}),
	)
	grpcServer := grpc.NewServer(grpcOptions...)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		logger.LoggerAgent.ErrorC(logging.PrintError(logging.Error1100, logging.BLOCKER, "Failed to listen on port: %v, error: %v", port, err.Error()))
	}
	apkmgt.RegisterEventStreamServiceServer(grpcServer, &managementserver.EventServer{})

	logger.LoggerAgent.Info("port: ", port, " APK agent Listening for gRPC connections")

	go managementserver.StartInternalServer(restPort)

	go func() {
		logger.LoggerAgent.Info("Starting GRPC server.")
		if err = grpcServer.Serve(lis); err != nil {
			logger.LoggerAgent.ErrorC(logging.PrintError(logging.Error1101, logging.BLOCKER, "Failed to start GRPC server, error: %v", err.Error()))
		}
	}()

	logger.LoggerAgent.Info("Bye!")
}
