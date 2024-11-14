package server

import (
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	pb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	"bigkinds.or.kr/pkg/server/service"
)

func NewGrpcServer(opts ...grpc.ServerOption) (server *grpc.Server) {
	opts = append(opts, grpc.KeepaliveParams(
		keepalive.ServerParameters{},
	))
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterHealthServer(grpcServer, &service.HealthServer{})
	reflection.Register(grpcServer)
	return grpcServer
}

func ServeGracefully(srv *grpc.Server, lis net.Listener, chanSig chan os.Signal) error {
	signal.Notify(chanSig, syscall.SIGINT, syscall.SIGTERM)

	chanSrvErr := make(chan error, 1)
	defer close(chanSrvErr)

	go func() {
		srvErr := srv.Serve(lis)
		if srvErr != nil {
			chanSrvErr <- srvErr
		}
	}()
	select {
	case <-chanSig: // if server is closed by signal
		srv.GracefulStop()
		return nil
	case srvErr := <-chanSrvErr: // if server is closed without signal
		return srvErr
	}
}
