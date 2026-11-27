package main

import (
	"context"
	"crypto/rsa"
	"flag"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/faust8888/go-musthave-metrics/internal/agent"
	"github.com/faust8888/go-musthave-metrics/internal/buildinfo"
	"github.com/faust8888/go-musthave-metrics/internal/config"
	"github.com/faust8888/go-musthave-metrics/internal/cryptokey"
	"github.com/faust8888/go-musthave-metrics/internal/envconfig"
)

var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

func main() {
	buildinfo.Print(buildVersion, buildDate, buildCommit)

	addr := flag.String("a", "localhost:8080", "HTTP server address")
	reportSec := flag.Int("r", 10, "report interval in seconds")
	pollSec := flag.Int("p", 2, "poll interval in seconds")
	key := flag.String("k", "", "hash key")
	rateLimit := flag.Int("l", 1, "max concurrent outgoing requests")
	cryptoKey := flag.String("crypto-key", "", "path to RSA public key PEM")
	grpcAddress := flag.String("g", "localhost:3200", "gRPC server address")
	grpcTLSCA := flag.String("grpc-tls-ca", "", "trusted gRPC server certificate (PEM)")
	flag.StringVar(grpcAddress, "grpc-address", "localhost:3200", "gRPC server address")
	cfgPath := ""
	flag.StringVar(&cfgPath, "c", "", "path to JSON config file")
	flag.StringVar(&cfgPath, "config", "", "path to JSON config file")
	flag.Parse()

	visited := config.Visited()
	envconfig.String("CONFIG", &cfgPath)

	var file config.AgentFile
	if err := config.Load(cfgPath, &file); err != nil {
		log.Fatal(err)
	}
	config.Apply(addr, file.Address, visited, "a")
	config.Apply(reportSec, file.ReportInterval.IntPtr(), visited, "r")
	config.Apply(pollSec, file.PollInterval.IntPtr(), visited, "p")
	config.Apply(key, file.Key, visited, "k")
	config.Apply(rateLimit, file.RateLimit, visited, "l")
	config.Apply(cryptoKey, file.CryptoKey, visited, "crypto-key")
	config.Apply(grpcAddress, file.GRPCAddress, visited, "g", "grpc-address")
	config.Apply(grpcTLSCA, file.GRPCTLSCA, visited, "grpc-tls-ca")
	envconfig.String("GRPC_TLS_CA", grpcTLSCA)

	envconfig.String("ADDRESS", addr)
	envconfig.String("KEY", key)
	envconfig.String("CRYPTO_KEY", cryptoKey)
	envconfig.String("GRPC_ADDRESS", grpcAddress)
	if err := envconfig.Int("REPORT_INTERVAL", reportSec); err != nil {
		log.Fatal(err)
	}
	if err := envconfig.Int("POLL_INTERVAL", pollSec); err != nil {
		log.Fatal(err)
	}
	if err := envconfig.Int("RATE_LIMIT", rateLimit); err != nil {
		log.Fatal(err)
	}

	pollInterval := time.Duration(*pollSec) * time.Second
	reportInterval := time.Duration(*reportSec) * time.Second

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	log.Printf("Agent started: grpc_server=%s poll=%ds report=%ds rate_limit=%d",
		*grpcAddress, *pollSec, *reportSec, *rateLimit)

	grpcAgent, err := agent.NewGRPC(*grpcAddress, *grpcTLSCA, pollInterval, reportInterval, *rateLimit)
	if err != nil {
		log.Fatal(err)
	}
	grpcAgent.Run(ctx)
	log.Println("Agent stopped")
}

func loadPublicKey(path string) (*rsa.PublicKey, error) {
	if path == "" {
		return nil, nil
	}
	return cryptokey.LoadPublic(path)
}
