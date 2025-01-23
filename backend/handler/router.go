package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"bigkinds.or.kr/backend/middleware/auth"
	"bigkinds.or.kr/backend/repository"
	"bigkinds.or.kr/backend/service"
	"bigkinds.or.kr/pkg/log"
	"github.com/go-chi/chi/v5"
	"github.com/segmentio/kafka-go"
	httpSwagger "github.com/swaggo/http-swagger"
	"gorm.io/gorm"

	_ "bigkinds.or.kr/backend/docs"
)

func NewRouter(db *gorm.DB, writer *kafka.Writer) chi.Router {
	koreanStandardTime, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		panic(err)
	}

	router := chi.NewRouter()
	authenticator := auth.Authenticator{
		AuthService: service.NewAuthService(),
	}

	apiRepository := repository.NewApiRepository(db, koreanStandardTime)
	chatRepository := repository.NewChatRepository(db)
	qARepository := repository.NewQARepository(db, koreanStandardTime)
	fileRepository := repository.NewFileRepository(db)

	apiService := &service.ApiService{
		ApiRepository: apiRepository,
	}

	chatService := &service.ChatService{
		ChatRepository: chatRepository,
		QARepository:   qARepository,
	}

	questionGuidesService := &service.QuestionGuidesService{}

	eventLogService := service.NewEventLogService(writer)
	fileService := &service.FileService{
		FileRepository: fileRepository,
		QARepository:   qARepository,
	}

	slog.Info(fmt.Sprintf("===== NewRouter -> q id: %T", eventLogService))

	completionService, err := service.NewCompletionService(
		chatService,
		eventLogService,
		fileService,
	)
	if err != nil {
		panic(err)
	}

	issueService, err := service.NewIssueService()
	if err != nil {
		panic(err)
	}

	newsService, err := service.NewNewsService()
	if err != nil {
		panic(err)
	}

	fileHandler := &FileHandler{
		UploadDir:         "./upload",
		MaxSize:           int64(30 * 1024 * 1024 * 1024 * 1024 * 1024),
		MaxNum:            int64(5),
		FileService:       fileService,
		ChatService:       chatService,
		CompletionService: completionService,
	}

	chatHandler := &ChatHandler{
		ChatService: chatService,
	}

	statisticsHandler := &StatisticsHandler{
		StatisticsService: &service.StatisticsService{
			Repository: qARepository,
		},
	}

	qaHandler := &qaHandler{
		service: &service.QAService{
			Repository: qARepository,
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
		Service: newsService,
	}

	apiHandler := &ApiHandler{
		ApiService: apiService,
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
			router.Get("/{chat_id}/qas", chatHandler.GetQAs)
			router.Get("/", chatHandler.GetUserChats)
			router.Post("/login", chatHandler.Login)
			router.Post("/", chatHandler.CreateUserChat)
			router.Post("/{chat_id}/completions/multi", completionHandler.CreateChatCompletionMulti)
			router.Post("/{chat_id}/completion/file", fileHandler.CreateChatCompletionFile)
		})
		router.Route("/issue", func(router chi.Router) {
			router.Use(authenticator.AuthMiddleware)
			router.Route("/topic", func(router chi.Router) {
				router.Post("/summary", issueHandler.GetIssueTopicSummary)
			})
		})
		router.Route("/news", func(router chi.Router) {
			router.Use(authenticator.AuthMiddleware)
			router.Post("/summary", newsHandler.GetNewsSummary)
		})

		router.Route("/file", func(router chi.Router) {
			router.Use(authenticator.AuthMiddleware)
			router.Post("/upload", fileHandler.FileUpload)
			router.Post("/upload-multiple/{chat_id}", fileHandler.MultipleFileUpload)
		})
	})

	router.Route("/api", func(router chi.Router) {
		router.Route("/v1", func(router chi.Router) {
			router.Use(authenticator.AuthMiddleware)
			router.Post("/apikey", apiHandler.CreateApikey)
			router.Get("/apikey/{apikey}", apiHandler.GetApikey)
			router.Put("/apikey", apiHandler.UpdateApikey)
			router.Delete("/apikey/{apikey}", apiHandler.DeleteApikey)
		})
	})

	router.Get("/healthz", func(responseWriter http.ResponseWriter, _ *http.Request) {
		responseWriter.WriteHeader(http.StatusOK)
		_, _ = responseWriter.Write([]byte("OK"))
	})

	router.Route("/dev", func(router chi.Router) {
		router.Use(authenticator.AuthMiddleware)
		router.Get("/uploadId/{chat_id}", fileHandler.GetUploadId)
	})

	router.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("http://gnew-biz.tplinkdns.com:8080/swagger/doc.json"),
	))

	return router
}
