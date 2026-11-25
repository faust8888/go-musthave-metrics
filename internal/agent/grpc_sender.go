package agent

import (
	"context"
	"fmt"

	models "github.com/faust8888/go-musthave-metrics/internal/model"
	metricspb "github.com/faust8888/go-musthave-metrics/internal/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// GRPCSender sends metric batches to a Metrics gRPC service.
type GRPCSender struct {
	address string
	conn    *grpc.ClientConn
	client  metricspb.MetricsClient
}

// NewGRPCSender creates a sender for address using plaintext transport.
func NewGRPCSender(address string) (*GRPCSender, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("create gRPC connection: %w", err)
	}
	return &GRPCSender{address: address, conn: conn, client: metricspb.NewMetricsClient(conn)}, nil
}

// SendBatch sends metrics in one UpdateMetrics RPC and includes the selected
// source IP in outgoing metadata for the server's trusted subnet check.
func (s *GRPCSender) SendBatch(ctx context.Context, metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}
	ip, err := localIPForServer(s.address)
	if err != nil {
		return fmt.Errorf("determine agent IP: %w", err)
	}
	request := &metricspb.UpdateMetricsRequest{Metrics: make([]*metricspb.Metric, 0, len(metrics))}
	for _, metric := range metrics {
		converted := &metricspb.Metric{Id: metric.ID}
		switch metric.MType {
		case models.Gauge:
			converted.Type = metricspb.Metric_GAUGE
			if metric.Value != nil {
				converted.Value = *metric.Value
			}
		case models.Counter:
			converted.Type = metricspb.Metric_COUNTER
			if metric.Delta != nil {
				converted.Delta = *metric.Delta
			}
		default:
			return fmt.Errorf("unsupported metric type %q", metric.MType)
		}
		request.Metrics = append(request.Metrics, converted)
	}
	ctx, cancel := context.WithTimeout(ctx, defaultHTTPTimeout)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "x-real-ip", ip)
	if _, err := s.client.UpdateMetrics(ctx, request); err != nil {
		return fmt.Errorf("update metrics over gRPC: %w", err)
	}
	return nil
}

// Close releases the underlying gRPC connection.
func (s *GRPCSender) Close() error { return s.conn.Close() }
