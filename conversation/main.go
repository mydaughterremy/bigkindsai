package main

import (
	"log"
	"net/http"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/joho/godotenv"

	"bigkinds.or.kr/conversation/handler"
	"bigkinds.or.kr/pkg/server"
)

var args struct {
	RestEndpoint string `arg:"--rest-endpoint" help:"REST endpoint to connect to" default:":8080"`
}

func main() {
	arg.MustParse(&args)
	_ = godotenv.Load()

	r := handler.NewRouter()
	log.Println("Starting server on", args.RestEndpoint)

	srv := &http.Server{
		Addr:    args.RestEndpoint,
		Handler: r,
	}
	chanSig := make(chan os.Signal, 1)

	if err := server.ListenAndServeGracefully(srv, chanSig); err != nil {
		panic(err)
	}
}
