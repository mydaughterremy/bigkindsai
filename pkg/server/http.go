package server

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	server_service "bigkinds.or.kr/pkg/server/service"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

func ListenAndServeGracefully(srv *http.Server, chanSig chan os.Signal) error {
	signal.Notify(chanSig, syscall.SIGINT, syscall.SIGTERM)

	chanSrvErr := make(chan error, 1)
	defer close(chanSrvErr)

	go func() {
		srvErr := srv.ListenAndServe()
		chanSrvErr <- srvErr
	}()
	select {
	case <-chanSig: // if server is closed by signal
		terminationGracePeriodSecondsString, ok := os.LookupEnv("TERMINATION_GRACE_PERIOD_SECONDS")
		if !ok {
			terminationGracePeriodSecondsString = "30"
		}
		terminationGracePeriodSeconds, err := strconv.ParseInt(terminationGracePeriodSecondsString, 10, 64)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(terminationGracePeriodSeconds)*time.Second) // shorter than kubernetes terminationGracePeriod(60s) - prestop (20s) = 40s
		defer cancel()
		if shutdownErr := srv.Shutdown(ctx); shutdownErr != nil {
			return shutdownErr
		}
		srvErr := <-chanSrvErr
		return srvErr
	case srvErr := <-chanSrvErr: // if server is closed without signal
		return srvErr
	}
}

func NewTenantServeMux() *runtime.ServeMux {
	mux := runtime.NewServeMux(
		runtime.WithHealthzEndpoint(
			&server_service.HealthClient{Server: &server_service.HealthServer{}},
		),
		runtime.WithIncomingHeaderMatcher(func(key string) (string, bool) {
			switch key {
			case "X-Upstage-Tenant-Id":
				return key, true
			default:
				return runtime.DefaultHeaderMatcher(key)
			}
		}),
	)
	return mux
}
