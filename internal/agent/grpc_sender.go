package agent

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"

	models "github.com/faust8888/go-musthave-metrics/internal/model"
	metricspb "github.com/faust8888/go-musthave-metrics/internal/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

// GRPCSender sends metric batches to a Metrics gRPC service.
type GRPCSender struct {
	conn   *grpc.ClientConn
	client metricspb.MetricsClient
	ip     string
}

// NewGRPCSender creates a TLS sender that trusts the certificate at caPath.
func NewGRPCSender(address, caPath string) (*GRPCSender, error) {
	caPEM, err := os.ReadFile(caPath)
	if err != nil {
		return nil, fmt.Errorf("read gRPC CA certificate: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("invalid gRPC CA certificate")
	}
	serverName, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("invalid gRPC address: %w", err)
	}
	if ip := net.ParseIP(serverName); ip != nil && ip.IsLoopback() {
		serverName = "localhost"
	}
	creds := credentials.NewTLS(&tls.Config{RootCAs: pool, ServerName: serverName, MinVersion: tls.VersionTLS12})
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("create gRPC connection: %w", err)
	}
	ip, err := localIPForServer(address)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("determine agent IP: %w", err)
	}
	return &GRPCSender{conn: conn, client: metricspb.NewMetricsClient(conn), ip: ip}, nil
}

// SendBatch sends metrics in one UpdateMetrics RPC and includes the selected
// source IP in outgoing metadata for the server's trusted subnet check.
func (s *GRPCSender) SendBatch(ctx context.Context, metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}
	request := &metricspb.UpdateMetricsRequest{}
	request.SetMetrics(make([]*metricspb.Metric, 0, len(metrics)))
	for _, metric := range metrics {
		converted := &metricspb.Metric{}
		converted.SetId(metric.ID)
		switch metric.MType {
		case models.Gauge:
			converted.SetType(metricspb.Metric_GAUGE)
			if metric.Value != nil {
				converted.SetValue(*metric.Value)
			}
		case models.Counter:
			converted.SetType(metricspb.Metric_COUNTER)
			if metric.Delta != nil {
				converted.SetDelta(*metric.Delta)
			}
		default:
			return fmt.Errorf("unsupported metric type %q", metric.MType)
		}
		request.SetMetrics(append(request.GetMetrics(), converted))
	}
	ctx, cancel := context.WithTimeout(ctx, defaultHTTPTimeout)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "x-real-ip", s.ip)
	if _, err := s.client.UpdateMetrics(ctx, request); err != nil {
		return fmt.Errorf("update metrics over gRPC: %w", err)
	}
	return nil
}

// Close releases the underlying gRPC connection.
func (s *GRPCSender) Close() error { return s.conn.Close() }
