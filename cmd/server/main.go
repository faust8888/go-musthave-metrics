package main

import (
	"context"
	"crypto/rsa"
	"database/sql"
	"flag"
	"log"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
	"google.golang.org/grpc/credentials"

	"github.com/faust8888/go-musthave-metrics/internal/audit"
	"github.com/faust8888/go-musthave-metrics/internal/buildinfo"
	"github.com/faust8888/go-musthave-metrics/internal/config"
	"github.com/faust8888/go-musthave-metrics/internal/cryptokey"
	"github.com/faust8888/go-musthave-metrics/internal/envconfig"
	"github.com/faust8888/go-musthave-metrics/internal/handler"
	"github.com/faust8888/go-musthave-metrics/internal/middleware"
	metricspb "github.com/faust8888/go-musthave-metrics/internal/proto"
	"github.com/faust8888/go-musthave-metrics/internal/repository"
	"github.com/faust8888/go-musthave-metrics/internal/service"
	"google.golang.org/grpc"
)

var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

func main() {
	buildinfo.Print(buildVersion, buildDate, buildCommit)

	addr := flag.String("a", "localhost:8080", "HTTP server address")
	storeInterval := flag.Int("i", 300, "store interval in seconds (0 = sync)")
	filePath := flag.String("f", "/tmp/metrics-db.json", "file storage path")
	restore := flag.Bool("r", true, "restore metrics from file on start")
	dsn := flag.String("d", "", "PostgreSQL DSN (DATABASE_DSN)")
	key := flag.String("k", "", "hash key")
	auditFile := flag.String("audit-file", "", "audit log file path")
	auditURL := flag.String("audit-url", "", "audit HTTP endpoint")
	cryptoKey := flag.String("crypto-key", "", "path to RSA private key PEM")
	trustedSubnet := flag.String("t", "", "trusted agent subnet in CIDR notation")
	grpcAddress := flag.String("g", "localhost:3200", "gRPC server address")
	grpcTLSCert := flag.String("grpc-tls-cert", "", "gRPC TLS certificate file")
	grpcTLSKey := flag.String("grpc-tls-key", "", "gRPC TLS private key file")
	flag.StringVar(grpcAddress, "grpc-address", "localhost:3200", "gRPC server address")
	cfgPath := ""
	flag.StringVar(&cfgPath, "c", "", "path to JSON config file")
	flag.StringVar(&cfgPath, "config", "", "path to JSON config file")
	flag.Parse()

	visited := config.Visited()
	envconfig.String("CONFIG", &cfgPath)

	var file config.ServerFile
	if err := config.Load(cfgPath, &file); err != nil {
		log.Fatal(err)
	}
	config.Apply(addr, file.Address, visited, "a")
	config.Apply(storeInterval, file.StoreInterval.IntPtr(), visited, "i")
	config.Apply(filePath, file.StoreFile, visited, "f")
	config.Apply(restore, file.Restore, visited, "r")
	config.Apply(dsn, file.DatabaseDSN, visited, "d")
	config.Apply(key, file.Key, visited, "k")
	config.Apply(auditFile, file.AuditFile, visited, "audit-file")
	config.Apply(auditURL, file.AuditURL, visited, "audit-url")
	config.Apply(cryptoKey, file.CryptoKey, visited, "crypto-key")
	config.Apply(trustedSubnet, file.TrustedSubnet, visited, "t")
	config.Apply(grpcAddress, file.GRPCAddress, visited, "g", "grpc-address")
	config.Apply(grpcTLSCert, file.GRPCTLSCert, visited, "grpc-tls-cert")
	config.Apply(grpcTLSKey, file.GRPCTLSKey, visited, "grpc-tls-key")

	envconfig.String("ADDRESS", addr)
	envconfig.String("STORE_FILE", filePath)
	envconfig.String("FILE_STORAGE_PATH", filePath)
	envconfig.String("DATABASE_DSN", dsn)
	envconfig.String("KEY", key)
	envconfig.String("AUDIT_FILE", auditFile)
	envconfig.String("AUDIT_URL", auditURL)
	envconfig.String("CRYPTO_KEY", cryptoKey)
	envconfig.String("TRUSTED_SUBNET", trustedSubnet)
	envconfig.String("GRPC_ADDRESS", grpcAddress)
	envconfig.String("GRPC_TLS_CERT", grpcTLSCert)
	envconfig.String("GRPC_TLS_KEY", grpcTLSKey)
	if err := envconfig.Int("STORE_INTERVAL", storeInterval); err != nil {
		log.Fatal(err)
	}
	if err := envconfig.Bool("RESTORE", restore); err != nil {
		log.Fatal(err)
	}
	var trustedIPNet *net.IPNet
	if *trustedSubnet != "" {
		_, parsedSubnet, err := net.ParseCIDR(*trustedSubnet)
		if err != nil {
			log.Fatalf("invalid trusted subnet %q: %v", *trustedSubnet, err)
		}
		trustedIPNet = parsedSubnet
	}

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	var db *sql.DB
	if *dsn != "" {
		db, err = sql.Open("pgx", *dsn)
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		if err = repository.RunMigrations(db); err != nil {
			log.Fatal("migrations failed: ", err)
		}
		logger.Info("database ready", zap.String("dsn", *dsn))
	}

	store, cleanup := newStorage(ctx, db, *filePath, *storeInterval, *restore, logger)
	defer cleanup()

	auditor, auditErr := audit.New(*auditFile, *auditURL)
	if auditErr != nil {
		log.Fatal("audit: ", auditErr)
	}
	defer auditor.Close()

	priv, err := loadPrivateKey(*cryptoKey)
	if err != nil {
		log.Fatal(err)
	}
	grpcListener, err := net.Listen("tcp", *grpcAddress)
	if err != nil {
		log.Fatalf("gRPC listen: %v", err)
	}
	grpcOptions := []grpc.ServerOption{}
	if trustedIPNet != nil {
		grpcOptions = append(grpcOptions, grpc.UnaryInterceptor(middleware.TrustedSubnetUnaryInterceptor(trustedIPNet)))
	}
	if *grpcTLSCert == "" || *grpcTLSKey == "" {
		log.Fatal("gRPC TLS certificate and key are required")
	}
	creds, err := credentials.NewServerTLSFromFile(*grpcTLSCert, *grpcTLSKey)
	if err != nil {
		log.Fatalf("gRPC TLS: %v", err)
	}
	grpcOptions = append(grpcOptions, grpc.Creds(creds))
	grpcServer := grpc.NewServer(grpcOptions...)
	metricspb.RegisterMetricsServer(grpcServer, service.NewGRPCMetricsService(store))
	go func() {
		if err := grpcServer.Serve(grpcListener); err != nil && err != grpc.ErrServerStopped {
			logger.Error("gRPC server stopped", zap.Error(err))
		}
	}()

	r := chi.NewRouter()
	r.Use(middleware.DecryptRSA(priv))
	r.Use(middleware.GzipDecompress)
	r.Use(middleware.GzipCompress)
	r.Use(middleware.Logger(logger))
	r.Use(middleware.HashSHA256(*key))

	r.Get("/ping", handler.Ping(db))
	r.Get("/", handler.Index(store))
	r.Group(func(r chi.Router) {
		if trustedIPNet != nil {
			r.Use(middleware.TrustedSubnet(trustedIPNet))
		}
		r.Post("/update/{type}/{name}/{value}", handler.Update(store, auditor))
		r.Post("/update", handler.UpdateJSON(store, auditor))
		r.Post("/update/", handler.UpdateJSON(store, auditor))
		r.Post("/updates/", handler.UpdatesBatch(store, auditor))
	})
	r.Get("/value/{type}/{name}", handler.Value(store))
	r.Post("/value", handler.ValueJSON(store))
	r.Post("/value/", handler.ValueJSON(store))

	srv := &http.Server{Addr: *addr, Handler: r}

	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		shutdownDone := make(chan struct{})
		go func() {
			defer close(shutdownDone)
			if err := srv.Shutdown(shutCtx); err != nil {
				logger.Error("HTTP shutdown error", zap.Error(err))
			}
		}()
		grpcDone := make(chan struct{})
		go func() { defer close(grpcDone); grpcServer.GracefulStop() }()
		select {
		case <-grpcDone:
		case <-shutCtx.Done():
			grpcServer.Stop()
		}
		select {
		case <-shutdownDone:
		case <-shutCtx.Done():
		}
		if err := shutCtx.Err(); err != nil {
			logger.Error("shutdown deadline exceeded", zap.Error(err))
		}
	}()

	logger.Info("Server started", zap.String("address", *addr), zap.String("grpc_address", *grpcAddress))
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	if path == "" {
		return nil, nil
	}
	return cryptokey.LoadPrivate(path)
}
