package server_test

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "google.golang.org/grpc/health/grpc_health_v1"

	"bigkinds.or.kr/pkg/server"
)

func TestNewGrpcServer(t *testing.T) {
	grpcServer := server.NewGrpcServer()
	assert.NotNil(t, grpcServer)
	grpcServer.Stop()
}

func TestServeGracefully(t *testing.T) {
	grpcServer := server.NewGrpcServer()

	lis, err := net.Listen("tcp", "localhost:0")
	assert.NoError(t, err)

	chanSig := make(chan os.Signal, 1)
	defer close(chanSig)

	go func() {
		time.Sleep(500 * time.Millisecond)
		chanSig <- os.Interrupt
	}()

	err = server.ServeGracefully(grpcServer, lis, chanSig)
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(
		ctx,
		lis.Addr().String(),
		grpc.WithBlock(),
		grpc.FailOnNonTempDialError(true),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	assert.Error(t, err)
	assert.Nil(t, conn)
}

func TestHealthCheck(t *testing.T) {
	grpcServer := server.NewGrpcServer()

	lis, err := net.Listen("tcp", "localhost:0")
	assert.NoError(t, err)

	go func() {
		err := grpcServer.Serve(lis)
		assert.NoError(t, err)
	}()

	conn, err := grpc.Dial(
		lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	assert.NoError(t, err)

	healthClient := pb.NewHealthClient(conn)
	healthReq := &pb.HealthCheckRequest{}
	healthRes, err := healthClient.Check(context.Background(), healthReq)
	assert.NoError(t, err)
	assert.Equal(t, pb.HealthCheckResponse_SERVING, healthRes.Status)

	grpcServer.Stop()
	conn.Close()
}
