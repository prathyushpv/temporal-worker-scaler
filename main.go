package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"strings"

	"github.com/prathyushpv/grpc-scaler/externalscaler"
	"go.temporal.io/sdk/client"
	sdklog "go.temporal.io/sdk/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// Server implements the ExternalScaler gRPC server.
type Server struct {
	externalscaler.UnimplementedExternalScalerServer
	client client.Client
}

func (s *Server) IsActive(ctx context.Context, scaledObjectRef *externalscaler.ScaledObjectRef) (*externalscaler.IsActiveResponse, error) {
	return &externalscaler.IsActiveResponse{Result: true}, nil
}

func (s *Server) GetMetricSpec(ctx context.Context, scaledObjectRef *externalscaler.ScaledObjectRef) (*externalscaler.MetricSpecResponse, error) {
	metricName := "taskqueue_backlog"

	// Absolute target value for the entire deployment, not per-replica. Scale up if it is greater than this value.
	targetValue := int64(100)

	metricSpec := &externalscaler.MetricSpec{
		MetricName:  metricName,
		TargetValue: targetValue, // Absolute value
	}

	log.Printf("Response of  GetMetricSpec: %v", metricSpec)
	return &externalscaler.MetricSpecResponse{
		MetricSpecs: []*externalscaler.MetricSpec{metricSpec},
	}, nil
}

func (s *Server) GetMetrics(ctx context.Context, req *externalscaler.GetMetricsRequest) (*externalscaler.GetMetricsResponse, error) {
	taskQueues := os.Getenv("TEMPORAL_TASK_QUEUES")
	taskQueueNames := strings.SplitN(taskQueues, ",", -1)

	var backlogCount int64
	for _, taskQueueName := range taskQueueNames {
		resp, err := s.client.DescribeTaskQueueEnhanced(ctx, client.DescribeTaskQueueEnhancedOptions{
			TaskQueue: taskQueueName,
		})
		if err != nil {
			log.Printf("Error describing task queue %s: %v", taskQueueName, err)
			return &externalscaler.GetMetricsResponse{}, err
		}

		// Get the backlog count from the enhanced response
		backlogCount += getBacklogCount(resp)
	}

	// Return the backlog count as the metric value (total backlog size)
	metric := &externalscaler.MetricValue{
		MetricName:  req.MetricName, // Use the requested metric name
		MetricValue: backlogCount,   // Return the backlog size as the metric value
	}

	log.Printf("Response of  GetMetrics: %v", metric)
	return &externalscaler.GetMetricsResponse{
		MetricValues: []*externalscaler.MetricValue{metric},
	}, nil
}

func getBacklogCount(description client.TaskQueueDescription) int64 {
	var count int64
	for _, versionInfo := range description.VersionsInfo {
		fmt.Printf("%v\n", versionInfo)
		for _, typeInfo := range versionInfo.TypesInfo {
			if typeInfo.Stats != nil {
				count += typeInfo.Stats.ApproximateBacklogCount
			}
		}
	}
	return count
}

func createClientOptionsFromEnv() (client.Options, error) {
	hostPort := os.Getenv("TEMPORAL_ADDRESS")
	namespaceName := os.Getenv("TEMPORAL_NAMESPACE")

	// Must explicitly set the Namepace for non-cloud use.
	if strings.Contains(hostPort, ".tmprl.cloud:") && namespaceName == "" {
		return client.Options{}, fmt.Errorf("Namespace name unspecified; required for Temporal Cloud")
	}

	if namespaceName == "" {
		namespaceName = "default"
		fmt.Printf("Namespace name unspecified; using value '%s'\n", namespaceName)
	}

	clientOpts := client.Options{
		HostPort:  hostPort,
		Namespace: namespaceName,
		Logger:    sdklog.NewStructuredLogger(slog.Default()),
	}

	if certPath := os.Getenv("TEMPORAL_TLS_CERT"); certPath != "" {
		cert, err := tls.LoadX509KeyPair(certPath, os.Getenv("TEMPORAL_TLS_KEY"))
		if err != nil {
			return clientOpts, fmt.Errorf("failed loading key pair: %w", err)
		}

		clientOpts.ConnectionOptions.TLS = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
	}

	return clientOpts, nil
}

func main() {
	// Start the gRPC server
	port := ":8080"
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}

	grpcServer := grpc.NewServer()
	clientOptions, err := createClientOptionsFromEnv()
	if err != nil {
		log.Printf("Failed to connect to temporal server")
		return
	}

	c, err := client.Dial(clientOptions)
	defer c.Close()
	externalscaler.RegisterExternalScalerServer(grpcServer, &Server{client: c})

	// Register reflection service on gRPC server.
	reflection.Register(grpcServer)

	log.Printf("gRPC server is running on port %s", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC server: %v", err)
	}
}
