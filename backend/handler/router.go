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
	kst, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		panic(err)
	}

	r := chi.NewRouter()
	authenticator := auth.Authenticator{
		AuthService: &service.AuthService{},
	}

	chatService := &service.ChatService{
		ChatRepository: repository.NewChatRepository(db),
		QARepository:   repository.NewQARepository(db, kst),
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

	chatHandler := &ChatHandler{
		ChatService: chatService,
	}

	statisticsHandler := &StatisticsHandler{
		StatisticsService: &service.StatisticsService{
			Repository: repository.NewQARepository(db, kst),
		},
	}

	qaHandler := &qaHandler{
		service: &service.QAService{
			Repository: repository.NewQARepository(db, kst),
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

	r.Use(log.RequestLogMiddleware)
	r.Use(log.ResponseLogMiddleware)

	r.Route("/v1", func(r chi.Router) {
		r.Route("/chats", func(r chi.Router) {
			r.Use(authenticator.AuthMiddleware)
			r.Post("/{chat_id}/completions", completionHandler.CreateChatCompletion)
			r.Get("/{chat_id}/qas", chatHandler.ListChatQAs)
			r.Post("/", chatHandler.CreateChat)
			r.Get("/", chatHandler.ListChats)
			r.Delete("/{chat_id}", chatHandler.DeleteChat)
			r.Put("/{chat_id}", chatHandler.UpdateChatTitle)
		})
		r.Route("/recommended_questions", func(r chi.Router) {
			r.Use(authenticator.AuthMiddleware)
			r.Get("/", GetRecommendedQuestions)
		})
		r.Route("/question-guides", func(r chi.Router) {
			r.Use(authenticator.AuthMiddleware)
			r.Get("/", questionGuidesHandler.GetQuestionGuides)
			r.Get("/tips", questionGuidesHandler.GetQuestionGuidesTips)
		})
		r.Route("/tutorial", func(r chi.Router) {
			r.Use(authenticator.AuthMiddleware)
			r.Get("/", tutorialHandler.GetTutorial)
		})
		r.Route("/statistics", func(r chi.Router) {
			r.Use(authenticator.AuthMiddleware)
			r.Get("/", statisticsHandler.GetStatistics)
		})
		r.Route("/qas", func(r chi.Router) {
			r.Use(authenticator.AuthMiddleware)
			r.Get("/", qaHandler.ListQAsWithPagination)
			r.Delete("/", qaHandler.DeleteQAs)
			r.Post("/delete", qaHandler.DeleteQAs)
			r.Route("/{qa_id}", func(r chi.Router) {
				r.Get("/", qaHandler.GetQA)
				r.Delete("/", qaHandler.DeleteQA)

				r.Route("/vote", func(r chi.Router) {
					r.Get("/", qaHandler.GetVote)
					r.Put("/up", qaHandler.UpvoteQA)
					r.Put("/down", qaHandler.DownvoteQA)
					r.Delete("/", qaHandler.DeleteVote)
				})
			})
		})
	})

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	return r
}
