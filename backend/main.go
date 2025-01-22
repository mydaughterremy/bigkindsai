package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/joho/godotenv"
	"github.com/segmentio/kafka-go"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"bigkinds.or.kr/backend/handler"
	"bigkinds.or.kr/backend/internal/db"
	"bigkinds.or.kr/backend/internal/server"
)

var args struct {
	RestEndpoint string `arg:"--rest-endpoint" help:"REST endpoint to connect to" default:":8080"`
}

func NewWriter() *kafka.Writer {
	brokers := strings.Split(os.Getenv("UPSTAGE_KAFKA_BROKERS"), ",")
	topic := os.Getenv("UPSTAGE_KAFKA_TOPIC")
	balancer := &kafka.LeastBytes{}

	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:      brokers,
		Topic:        topic,
		Balancer:     balancer,
		BatchTimeout: 100 * time.Millisecond,
		WriteTimeout: 100 * time.Millisecond,
		MaxAttempts:  3,
		BatchSize:    100,
		Async:        true,
		RequiredAcks: int(kafka.RequireOne),
	})

	writer.Completion = func(messages []kafka.Message, err error) {
		if err != nil {
			slog.Error("failed to write messages:", "error", err)
		}
	}

	return writer
}

// @title Bigkinds AI
// @ version 2.0
// @description This API for Bigkinds AI web service.
// @host gnew-biz.tplinkdns.com:8080

func main() {
	arg.MustParse(&args)
	_ = godotenv.Load()

	// open mysql connection
	mysqlDSN := db.CreateMySQLDSN(
		os.Getenv("UPSTAGE_MYSQL_HOST"),
		os.Getenv("UPSTAGE_MYSQL_PORT"),
		os.Getenv("UPSTAGE_MYSQL_USER"),
		os.Getenv("UPSTAGE_MYSQL_PASSWORD"),
		os.Getenv("UPSTAGE_MYSQL_DB"),
	)
	mysqlDB, err := gorm.Open(mysql.Open(mysqlDSN), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	// mysqlDB.AutoMigrate(&model.File{})
	// mysqlDB.AutoMigrate(&model.QA{})

	writer := NewWriter()
	defer writer.Close()

	router := handler.NewRouter(mysqlDB, writer)
	log.Println("Starting server on", args.RestEndpoint)

	restServer := &http.Server{
		Addr:    args.RestEndpoint,
		Handler: router,
	}
	channelSignal := make(chan os.Signal, 1)

	if err := server.ListenAndServeGracefully(restServer, channelSignal); err != nil {
		panic(err)
	}
}
