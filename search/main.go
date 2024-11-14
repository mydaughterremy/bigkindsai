package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"bigkinds.or.kr/pkg/log"
	"bigkinds.or.kr/pkg/reranker/e5"
	"bigkinds.or.kr/pkg/search_engine"
	"bigkinds.or.kr/pkg/server"
	server_service "bigkinds.or.kr/pkg/server/service"
	"bigkinds.or.kr/proto"
	"github.com/alexflint/go-arg"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"bigkinds.or.kr/search/repository/search_config"
	"bigkinds.or.kr/search/repository/searcher_config"
	"bigkinds.or.kr/search/service"
	"bigkinds.or.kr/search/service/provider"
	"bigkinds.or.kr/search/service/searcher/opensearch"
)

type Args struct {
	RestEndpoint string `arg:"-r,--rest-server-endpoint" default:"0.0.0.0:8080" help:"rest server endpoint"`
	GrpcEndpoint string `arg:"-g,--grpc-server-endpoint" default:"0.0.0.0:8081" help:"grpc server endpoint"`
}

func NewFileSearcherConfigRepository() (*searcher_config.FileSearcherConfigRepository, error) {
	return &searcher_config.FileSearcherConfigRepository{
		FilePath: filepath.Join("config", "searcher_config.json"),
	}, nil
}

func NewFileSearchConfigRepository() (*search_config.FileSearchConfigRepository, error) {
	return &search_config.FileSearchConfigRepository{
		FilePath: filepath.Join("config", "search_config.json"),
	}, nil
}

func initializeSingletonClient(logger *zap.Logger) {
	var err error

	// create E5RerankerClient
	_, err = e5.InitializeSingletonClient()
	if err != nil {
		fatalMsg := fmt.Sprintf("error initializing e5 reranker client: %v", err)
		logger.Fatal(fatalMsg)
	}

}

func main() {
	var args Args
	arg.MustParse(&args)

	err := godotenv.Load()
	if err != nil {
		fmt.Println(err)
	}

	logger, err := log.NewLogger("search")
	if err != nil {
		panic(err)
	}

	s := server.NewGrpcServer(
		grpc.ChainUnaryInterceptor(
			log.ContextLoggerUnaryServerInterceptor(logger),
			log.RequestLogUnaryServerInterceptor(),
			log.ResponseLogUnaryServerInterceptor(),
		),
	)

	lis, err := net.Listen("tcp", args.GrpcEndpoint)
	if err != nil {
		panic(err)
	}
	// initialize singleton client
	initializeSingletonClient(logger)

	// initialize openSearchClient
	addressString := os.Getenv("UPSTAGE_OPENSEARCH_ADDRESS")
	username := os.Getenv("UPSTAGE_OPENSEARCH_USERNAME")
	password := os.Getenv("UPSTAGE_OPENSEARCH_PASSWORD")
	_, err = search_engine.InitializeSingletonOpenSearchClient(addressString, username, password)
	if err != nil {
		fatalMsg := fmt.Sprintf("error initializing opensearch client: %v", err)
		logger.Fatal(fatalMsg)
	}

	if err != nil {
		fatalMsg := fmt.Sprintf("error initializing opensearch client: %v", err)
		logger.Fatal(fatalMsg)
	}

	// create rerankerProvider
	rerankerProvider, err := provider.NewRerankerProvider()
	if err != nil {
		panic(err)
	}

	// create ConfigmapSearchConfigRepository
	searchRepo, err := NewFileSearchConfigRepository()
	if err != nil {
		panic(err)
	}
	// get search config
	searchConfig, err := searchRepo.GetConfig(context.Background())
	if err != nil {
		panic(err)
	}

	// create searcherProvider
	queryStrategyProvider, err := opensearch.NewQueryStrategyProvider()
	if err != nil {
		panic(err)
	}

	searcherRepo, err := NewFileSearcherConfigRepository()
	if err != nil {
		panic(err)
	}
	// get searcher config
	searcherConfig, err := searcherRepo.GetConfig(context.Background())
	if err != nil {
		panic(err)
	}

	searcherProvider, err := provider.NewSearcherProvider(
		queryStrategyProvider,
		searcherConfig,
	)
	if err != nil {
		panic(err)
	}

	serviceServer := &service.SearchServiceServer{
		SearcherProvider: searcherProvider,
		RerankerProvider: rerankerProvider,
		SearchConfig:     searchConfig,
	}

	proto.RegisterSearchServiceServer(s, serviceServer)

	go func() {
		sigchan := make(chan os.Signal, 1)

		logger.Info(fmt.Sprintf("gRPC server listening at %s", lis.Addr()))
		if err := server.ServeGracefully(s, lis, sigchan); err != nil {
			fmt.Printf("failed to serve: %v", err.Error())
		}
	}()

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux(
		runtime.WithHealthzEndpoint(
			&server_service.HealthClient{Server: &server_service.HealthServer{}},
		),
	)
	err = proto.RegisterSearchServiceHandlerServer(ctx, mux, serviceServer)
	if err != nil {
		logger.Panic(err.Error())
	}

	srv := &http.Server{
		Addr:    args.RestEndpoint,
		Handler: mux,
	}

	logger.Info(fmt.Sprintf("HTTP server listening at %v", args.RestEndpoint))
	chanSigHTTP := make(chan os.Signal, 1)
	if err := server.ListenAndServeGracefully(srv, chanSigHTTP); err != nil {
		logger.Error(fmt.Sprintf("failed to serve: %v", err))
	}
}
