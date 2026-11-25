package service

import (
	"context"
	"fmt"

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

// NewGRPCMetricsService constructs a gRPC metrics service backed by store.
func NewGRPCMetricsService(store repository.Storage) *GRPCMetricsService {
	return &GRPCMetricsService{store: store}
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
	if err := s.store.UpdateBatch(ctx, metrics); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("store metrics: %v", err))
	}
	return &proto.UpdateMetricsResponse{}, nil
}
