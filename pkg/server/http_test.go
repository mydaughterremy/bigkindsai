package server

import (
	"net/http"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGracefulShutdown(t *testing.T) {

	chanSrvSig := make(chan os.Signal, 1)
	chanSrvErr := make(chan error, 2)
	defer close(chanSrvSig)
	defer close(chanSrvErr)

	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(time.Second * 2)
			w.WriteHeader(http.StatusOK)
		})

		srv := &http.Server{
			Addr:    "0.0.0.0:4567",
			Handler: mux,
		}

		err := ListenAndServeGracefully(srv, chanSrvSig)
		chanSrvErr <- err
	}()

	time.Sleep(time.Second * 2) // wait to open server

	chanReqErr := make(chan error, 1)
	defer close(chanReqErr)
	go func() {
		_, err := http.Get("http://localhost:4567/")
		chanReqErr <- err
	}()

	time.Sleep(time.Second * 1) // wait for connection
	chanSrvSig <- syscall.SIGTERM

	select {
	case err := <-chanSrvErr:
		assert.Error(t, http.ErrServerClosed, err, "check if is closed")
	case <-time.After(30 * time.Second):
		assert.Fail(t, "server close timeout")
	}

	select {
	case err := <-chanReqErr:
		assert.Nil(t, err, "check request error")
	case <-time.After(3 * time.Second):
		assert.Fail(t, "request timeout")
	}
}
