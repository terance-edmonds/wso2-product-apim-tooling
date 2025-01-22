package agent

import (
	"flag"
	"fmt"
	"net"
	"time"

	healthservice "github.com/wso2/apk/adapter/pkg/health/api/wso2/health/service"
	logger "github.com/wso2/product-apim-tooling/apim-agent/internal/loggers"
	logging "github.com/wso2/product-apim-tooling/apim-agent/internal/logging"
	"github.com/wso2/product-apim-tooling/apim-agent/pkg/health"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

var (
	grpcPort uint
)

func init() {
	flag.UintVar(&grpcPort, "grpcPort", 19000, "APIM Management server port")
}

// RunGRPCServer starts a grpc server for the health service
func RunGRPCServer() {
	var grpcOptions []grpc.ServerOption
	grpcOptions = append(grpcOptions, grpc.KeepaliveParams(
		keepalive.ServerParameters{
			Time:    time.Duration(5 * time.Minute),
			Timeout: time.Duration(20 * time.Second),
		}),
	)
	grpcServer := grpc.NewServer(grpcOptions...)
	// register health service
	healthservice.RegisterHealthServer(grpcServer, &health.Server{})
	logger.LoggerAgent.Info("port: ", grpcPort, " APIM agent Listening for gRPC connections")
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if err != nil {
		logger.LoggerAgent.ErrorC(logging.PrintError(logging.Error1100, logging.BLOCKER, "Failed to listen on port: %v, error: %v", grpcPort, err.Error()))
	}
	go func() {
		logger.LoggerAgent.Info("Starting GRPC server.")
		health.ApimAgentGrpcService.SetStatus(true)
		if err = grpcServer.Serve(lis); err != nil {
			health.ApimAgentGrpcService.SetStatus(false)
			logger.LoggerAgent.ErrorC(logging.PrintError(logging.Error1101, logging.BLOCKER, "Failed to start GRPC server, error: %v", err.Error()))
		}
	}()
}
