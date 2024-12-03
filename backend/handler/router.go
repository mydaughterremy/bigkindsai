package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/segmentio/kafka-go"

	"gorm.io/gorm"

	"bigkinds.or.kr/backend/middleware/auth"
	"bigkinds.or.kr/backend/repository"
	"bigkinds.or.kr/backend/service"

	"bigkinds.or.kr/pkg/log"
)

func NewRouter(db *gorm.DB, writer *kafka.Writer) chi.Router {
	koreanStandardTime, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		panic(err)
	}

	router := chi.NewRouter()
	authenticator := auth.Authenticator{
		AuthService: &service.AuthService{},
	}

	chatService := &service.ChatService{
		ChatRepository: repository.NewChatRepository(db),
		QARepository:   repository.NewQARepository(db, koreanStandardTime),
	}

	questionGuidesService := &service.QuestionGuidesService{}

	eventLogService := service.NewEventLogService(writer)

	completionService, err := service.NewCompletionService(
		chatService,
		eventLogService,
	)
	if err != nil {
		panic(err)
	}

	issueService, err := service.NewIssueService()
	if err != nil {
		panic(err)
	}

	chatHandler := &ChatHandler{
		ChatService: chatService,
	}

	statisticsHandler := &StatisticsHandler{
		StatisticsService: &service.StatisticsService{
			Repository: repository.NewQARepository(db, koreanStandardTime),
		},
	}

	qaHandler := &qaHandler{
		service: &service.QAService{
			Repository: repository.NewQARepository(db, koreanStandardTime),
		},
	}

	completionHandler := &completionHandler{
		service: completionService,
	}

	questionGuidesHandler := &questionGuidesHandler{
		service: questionGuidesService,
	}

	tutorialHandler := &tutorialHandler{
		service: &service.TutorialService{},
	}

	issueHandler := &IssueHandler{
		Service: issueService,
	}

	newsHandler := &NewsHandler{
		Service: &service.NewsService{},
	}

	router.Use(log.RequestLogMiddleware)
	router.Use(log.ResponseLogMiddleware)

	router.Route("/v1", func(router chi.Router) {
		router.Route("/chats", func(router chi.Router) {
			router.Use(authenticator.AuthMiddleware)
			router.Post("/{chat_id}/completions", completionHandler.CreateChatCompletion)
			router.Get("/{chat_id}/qas", chatHandler.ListChatQAs)
			router.Post("/", chatHandler.CreateChat)
			router.Get("/", chatHandler.ListChats)
			router.Delete("/{chat_id}", chatHandler.DeleteChat)
			router.Put("/{chat_id}", chatHandler.UpdateChatTitle)
		})
		router.Route("/recommended_questions", func(router chi.Router) {
			router.Use(authenticator.AuthMiddleware)
			router.Get("/", GetRecommendedQuestions)
		})
		router.Route("/question-guides", func(router chi.Router) {
			router.Use(authenticator.AuthMiddleware)
			router.Get("/", questionGuidesHandler.GetQuestionGuides)
			router.Get("/tips", questionGuidesHandler.GetQuestionGuidesTips)
		})
		router.Route("/tutorial", func(router chi.Router) {
			router.Use(authenticator.AuthMiddleware)
			router.Get("/", tutorialHandler.GetTutorial)
		})
		router.Route("/statistics", func(router chi.Router) {
			router.Use(authenticator.AuthMiddleware)
			router.Get("/", statisticsHandler.GetStatistics)
		})
		router.Route("/qas", func(router chi.Router) {
			router.Use(authenticator.AuthMiddleware)
			router.Get("/", qaHandler.ListQAsWithPagination)
			router.Delete("/", qaHandler.DeleteQAs)
			router.Post("/delete", qaHandler.DeleteQAs)
			router.Route("/{qa_id}", func(router chi.Router) {
				router.Get("/", qaHandler.GetQA)
				router.Delete("/", qaHandler.DeleteQA)

				router.Route("/vote", func(router chi.Router) {
					router.Get("/", qaHandler.GetVote)
					router.Put("/up", qaHandler.UpvoteQA)
					router.Put("/down", qaHandler.DownvoteQA)
					router.Delete("/", qaHandler.DeleteVote)
				})
			})
		})
	})

	router.Route("/v2", func(router chi.Router) {
		router.Route("/chats", func(router chi.Router) {
			router.Use(authenticator.AuthMiddleware)
			router.Get("/", chatHandler.GetUserChats)
			router.Post("/login", chatHandler.Login)
			router.Post("/", chatHandler.CreateUserChat)
			router.Post("/{chat_id}/completions/multi", completionHandler.CreateChatCompletionMulti)
		})
		router.Route("/issue", func(router chi.Router) {
			router.Use(authenticator.AuthMiddleware)
			router.Route("/topic", func(router chi.Router) {
				router.Get("/summary", issueHandler.GetIssueTopicSummary)
			})
		})
		router.Route("/news", func(router chi.Router) {
			router.Use(authenticator.AuthMiddleware)
			router.Post("/summary", newsHandler.GetNewsSummary)
		})
	})

	router.Get("/healthz", func(responseWriter http.ResponseWriter, _ *http.Request) {
		responseWriter.WriteHeader(http.StatusOK)
		_, _ = responseWriter.Write([]byte("OK"))
	})

	return router
}
