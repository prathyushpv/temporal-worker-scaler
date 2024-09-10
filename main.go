package main

import (
	"context"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/prathyushpv/grpc-scaler/externalscaler" // Import the generated proto package
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// Server implements the ExternalScaler gRPC server.
type Server struct {
	externalscaler.UnimplementedExternalScalerServer
}

// Add a global or persistent variable to store the previous value
var previousValue int

//func (s *Server) IsActive(ctx context.Context, scaledObjectRef *externalscaler.ScaledObjectRef) (*externalscaler.IsActiveResponse, error) {
//	// Read the current value from the file
//	filePath := "/data/command_output.txt"
//	fileContent, err := os.ReadFile(filePath)
//	if err != nil {
//		log.Printf("Error reading file %s: %v", filePath, err)
//		return &externalscaler.IsActiveResponse{Result: false}, nil // Return false if there's an error
//	}
//
//	trimmedContent := strings.TrimSpace(string(fileContent))
//	currentValue, err := strconv.Atoi(trimmedContent)
//	if err != nil {
//		log.Printf("Error converting file content to integer: %v", err)
//		return &externalscaler.IsActiveResponse{Result: false}, nil // Return false on conversion failure
//	}
//
//	// Determine if scaling is required based on the current and previous values
//	isActive := currentValue > previousValue // Active if current value is greater than previous
//
//	// Update the previous value for future comparison
//	previousValue = currentValue
//
//	fmt.Printf("IsActive: %v\n", isActive)
//	// Return whether scaling is active
//	return &externalscaler.IsActiveResponse{Result: isActive}, nil
//}

func (s *Server) IsActive(ctx context.Context, scaledObjectRef *externalscaler.ScaledObjectRef) (*externalscaler.IsActiveResponse, error) {
	return &externalscaler.IsActiveResponse{Result: true}, nil
}

//// GetMetricSpec defines the metric that will be used for scaling.
//func (s *Server) GetMetricSpec(ctx context.Context, scaledObjectRef *externalscaler.ScaledObjectRef) (*externalscaler.MetricSpecResponse, error) {
//	metricName := "custom_metric"
//	targetValue := int64(1000) // Define the target value for the metric.
//
//	metricSpec := &externalscaler.MetricSpec{
//		MetricName:  metricName,
//		TargetValue: targetValue,
//	}
//	fmt.Printf("GetMetricSpec: %v\n", metricSpec)
//
//	return &externalscaler.MetricSpecResponse{
//		MetricSpecs: []*externalscaler.MetricSpec{metricSpec},
//	}, nil
//}

func (s *Server) GetMetricSpec(ctx context.Context, scaledObjectRef *externalscaler.ScaledObjectRef) (*externalscaler.MetricSpecResponse, error) {
	metricName := "file_based_metric"

	// Absolute target value for the entire deployment, not per-replica
	targetValue := int64(1000)

	metricSpec := &externalscaler.MetricSpec{
		MetricName:  metricName,
		TargetValue: targetValue, // Absolute value
	}

	return &externalscaler.MetricSpecResponse{
		MetricSpecs: []*externalscaler.MetricSpec{metricSpec},
	}, nil
}

//// GetMetrics returns the actual value of the custom metric.
//func (s *Server) GetMetrics(ctx context.Context, req *externalscaler.GetMetricsRequest) (*externalscaler.GetMetricsResponse, error) {
//	filePath := "/data/command_output.txt"
//	fileContent, err := os.ReadFile(filePath)
//	if err != nil {
//		log.Printf("Error reading file %s: %v", filePath, err)
//		return &externalscaler.GetMetricsResponse{}, nil
//	}
//
//	trimmedContent := strings.TrimSpace(string(fileContent))
//	fileValue, err := strconv.Atoi(trimmedContent)
//	if err != nil {
//		log.Printf("Error converting file content to integer: %v", err)
//		return &externalscaler.GetMetricsResponse{}, nil
//	}
//
//	metricValue := &externalscaler.MetricValue{
//		MetricName:  req.MetricName,
//		MetricValue: int64(fileValue), // Scaling logic, e.g., divide by 1000
//	}
//
//	fmt.Printf("GetMetrics: %v\n", metricValue)
//	return &externalscaler.GetMetricsResponse{
//		MetricValues: []*externalscaler.MetricValue{metricValue},
//	}, nil
//}

func (s *Server) GetMetrics(ctx context.Context, req *externalscaler.GetMetricsRequest) (*externalscaler.GetMetricsResponse, error) {
	// Read the value from the file
	filePath := "/data/command_output.txt"
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("Error reading file %s: %v", filePath, err)
		return &externalscaler.GetMetricsResponse{}, nil // Return empty response on error
	}

	trimmedContent := strings.TrimSpace(string(fileContent))
	currentValue, err := strconv.Atoi(trimmedContent)
	if err != nil {
		log.Printf("Error converting file content to integer: %v", err)
		return &externalscaler.GetMetricsResponse{}, nil // Return empty response on error
	}

	// Divide the value by 1000 to calculate the absolute metric value
	metricValue := int64(currentValue)

	// Return the absolute value (not an average)
	metric := &externalscaler.MetricValue{
		MetricName:  req.MetricName, // Use the requested metric name
		MetricValue: metricValue,    // Return the total absolute value
	}

	return &externalscaler.GetMetricsResponse{
		MetricValues: []*externalscaler.MetricValue{metric},
	}, nil
}

func main() {
	// Start the gRPC server
	port := ":8080"
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}

	grpcServer := grpc.NewServer()
	externalscaler.RegisterExternalScalerServer(grpcServer, &Server{})

	// Register reflection service on gRPC server.
	reflection.Register(grpcServer)

	log.Printf("gRPC server is running on port %s", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC server: %v", err)
	}
}
