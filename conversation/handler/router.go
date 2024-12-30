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
	router := chi.NewRouter()

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
	topicService := service.NewTopicService()

	summaryService := service.NewSummaryService()

	completionHandler := &completionHandler{
		service: completionService,
	}
	topicHandler := &topicHandler{
		service: topicService,
	}

	completionMultiService := service.NewCompletionMultiService(functionService, tokenCounter)
	completionMultiHandler := &completionMultiHandler{
		s: completionMultiService,
	}

	summaryHandler := &summaryHandler{
		service: summaryService,
	}

	router.Use(log.RequestLogMiddleware)
	router.Use(log.ResponseLogMiddleware)

	router.Route("/v1", func(router chi.Router) {
		router.Post("/chat/completions", completionHandler.CreateChatCompletion)
		router.Post("/chat/completions/multi", completionMultiHandler.CreateChatCompletionMulti)
	})
	router.Route("/v2", func(router chi.Router) {
		router.Post("/topic", topicHandler.HandleTopic)
		router.Post("/summary", summaryHandler.SummaryContent)

	})

	router.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	return router
}
