package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pkoukk/tiktoken-go"

	"bigkinds.or.kr/conversation/internal/token_counter"
	service "bigkinds.or.kr/conversation/service"
	"bigkinds.or.kr/conversation/service/function"
	"bigkinds.or.kr/pkg/log"
)

func NewRouter() chi.Router {
	r := chi.NewRouter()

	functionService := &function.FunctionService{}

	tokenizer, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		panic(err)
	}
	tokenCounter := token_counter.NewTokenCounter(
		tokenizer,
	)

	completionService := service.NewCompletionService(
		functionService,
		tokenCounter,
	)

	completionHandler := &completionHandler{
		service: completionService,
	}

	r.Use(log.RequestLogMiddleware)
	r.Use(log.ResponseLogMiddleware)

	r.Route("/v1", func(r chi.Router) {
		r.Post("/chat/completions", completionHandler.CreateChatCompletion)
	})

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	return r
}
