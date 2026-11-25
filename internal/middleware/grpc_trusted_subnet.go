package middleware

import (
	"context"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// TrustedSubnetUnaryInterceptor rejects RPCs whose x-real-ip metadata is
// missing, invalid, or outside subnet. A nil subnet disables the check.
func TrustedSubnetUnaryInterceptor(subnet *net.IPNet) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if subnet == nil {
			return handler(ctx, req)
		}
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok || len(md.Get("x-real-ip")) == 0 {
			return nil, status.Error(codes.PermissionDenied, "agent IP is outside trusted subnet")
		}
		ip := net.ParseIP(md.Get("x-real-ip")[0])
		if ip == nil || !subnet.Contains(ip) {
			return nil, status.Error(codes.PermissionDenied, "agent IP is outside trusted subnet")
		}
		return handler(ctx, req)
	}
}
