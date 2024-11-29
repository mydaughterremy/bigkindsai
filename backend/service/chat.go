package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"bigkinds.or.kr/backend/model"
	"bigkinds.or.kr/backend/repository"
)

type ChatService struct {
	ChatRepository *repository.ChatRepository
	QARepository   *repository.QARepository
}

func (s *ChatService) GetChat(ctx context.Context, id uuid.UUID) (*model.Chat, error) {
	chat, err := s.ChatRepository.GetChat(ctx, id)
	if err != nil {
		return nil, err
	}
	return chat, nil
}

func (s *ChatService) CreateChat(ctx context.Context, sessionId string) (*model.Chat, error) {
	id := uuid.New()
	chat := &model.Chat{
		ID:        id,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Object:    "chat",
		Title:     "새로운 채팅",
		SessionID: sessionId,
	}

	err := s.ChatRepository.CreateChat(ctx, chat)
	if err != nil {
		return nil, err
	}

	return chat, nil
}

func (s *ChatService) ListChats(ctx context.Context, sessionId string) ([]*model.Chat, error) {
	chats, err := s.ChatRepository.ListChats(ctx, sessionId)
	if err != nil {
		return nil, err
	}

	return chats, nil
}

func (s *ChatService) DeleteChat(ctx context.Context, id uuid.UUID) error {
	err := s.ChatRepository.DeleteChat(ctx, id)
	if err != nil {
		return err
	}
	return nil
}

func (s *ChatService) UpdateChatTitle(ctx context.Context, id uuid.UUID, title string) (*model.Chat, error) {
	chat, err := s.ChatRepository.UpdateChat(ctx, &model.Chat{
		ID:    id,
		Title: title,
	})
	if err != nil {
		return nil, err
	}
	return chat, nil
}

func (s *ChatService) ListChatQAs(ctx context.Context, session, chatID string) ([]*model.QA, error) {
	qas, err := s.QARepository.ListChatQAs(ctx, session, chatID)
	if err != nil {
		return nil, err
	}
	return qas, nil
}

func (s *ChatService) LastChatQA(ctx context.Context, chatID string) (*model.QA, error) {
	qa, err := s.QARepository.LastChatQA(ctx, chatID)
	if err != nil {
		return nil, err
	}
	return qa, nil
}
