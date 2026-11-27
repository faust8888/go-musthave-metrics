package service

import (
	"context"
	"errors"
	"fmt"
	"log"

	models "github.com/faust8888/go-musthave-metrics/internal/model"
	"github.com/faust8888/go-musthave-metrics/internal/proto"
	"github.com/faust8888/go-musthave-metrics/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCMetricsService stores metric batches received over gRPC.
type GRPCMetricsService struct {
	proto.UnimplementedMetricsServer
	store repository.Storage
}

// ValidationError identifies invalid metric batches supplied by a transport.
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

// NewGRPCMetricsService constructs a gRPC metrics service backed by store.
func NewGRPCMetricsService(store repository.Storage) *GRPCMetricsService {
	return &GRPCMetricsService{store: store}
}

// ValidateMetrics applies shared metric rules before any transport stores a batch.
func ValidateMetrics(metrics []models.Metrics) error {
	for _, metric := range metrics {
		if metric.ID == "" {
			return &ValidationError{Message: "metric ID is required"}
		}
		switch metric.MType {
		case models.Gauge:
			if metric.Value == nil {
				return &ValidationError{Message: "value is required for gauge"}
			}
		case models.Counter:
			if metric.Delta == nil {
				return &ValidationError{Message: "delta is required for counter"}
			}
		default:
			return &ValidationError{Message: fmt.Sprintf("invalid metric type: %s", metric.MType)}
		}
	}
	return nil
}

// StoreMetrics validates and writes a batch for any transport.
func StoreMetrics(ctx context.Context, store repository.Storage, metrics []models.Metrics) error {
	if err := ValidateMetrics(metrics); err != nil {
		return err
	}
	return store.UpdateBatch(ctx, metrics)
}

// UpdateMetrics validates and stores a batch of metrics.
func (s *GRPCMetricsService) UpdateMetrics(ctx context.Context, req *proto.UpdateMetricsRequest) (*proto.UpdateMetricsResponse, error) {
	metrics := make([]models.Metrics, 0, len(req.GetMetrics()))
	for i, metric := range req.GetMetrics() {
		if metric == nil || metric.GetId() == "" {
			return nil, status.Errorf(codes.InvalidArgument, "metric %d has no ID", i)
		}
		converted := models.Metrics{ID: metric.GetId()}
		switch metric.GetType() {
		case proto.Metric_GAUGE:
			value := metric.GetValue()
			converted.MType = models.Gauge
			converted.Value = &value
		case proto.Metric_COUNTER:
			delta := metric.GetDelta()
			converted.MType = models.Counter
			converted.Delta = &delta
		default:
			return nil, status.Errorf(codes.InvalidArgument, "metric %q has unknown type %d", metric.GetId(), metric.GetType())
		}
		metrics = append(metrics, converted)
	}
	if err := StoreMetrics(ctx, s.store, metrics); err != nil {
		var validationErr *ValidationError
		if errors.As(err, &validationErr) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		log.Printf("store metrics: %v", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &proto.UpdateMetricsResponse{}, nil
}
