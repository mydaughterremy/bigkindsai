package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"bigkinds.or.kr/backend/model"
)

type ChatRepository struct {
	db *gorm.DB
}

func NewChatRepository(db *gorm.DB) *ChatRepository {
	return &ChatRepository{
		db: db,
	}
}

func (r *ChatRepository) GetChat(ctx context.Context, id uuid.UUID) (*model.Chat, error) {
	var chat model.Chat
	result := r.db.First(&model.Chat{}, id)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &chat, result.Error
}

func (r *ChatRepository) CreateChat(ctx context.Context, chat *model.Chat) error {
	result := r.db.Create(chat)
	return result.Error
}

func (r *ChatRepository) UpdateChat(ctx context.Context, Id uuid.UUID, title string) (*model.Chat, error) {
	chat := &model.Chat{}
	result := r.db.Model(chat).Clauses(clause.Returning{}).Where("id = ?", Id.String()).Update("title", title)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	chat.ID = Id

	return chat, nil
}

func (r *ChatRepository) UpdateChatUser(ctx context.Context, chat *model.Chat) (*model.Chat, error) {
	res := r.db.WithContext(ctx).Model(chat).Updates(
		model.Chat{
			UserHash: chat.UserHash,
		},
	)

	if res.Error != nil {
		return nil, res.Error
	}

	if res.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	return chat, nil
}

func (r *ChatRepository) ListChats(ctx context.Context, sessionId string) ([]*model.Chat, error) {
	var chats []*model.Chat
	result := r.db.Where("session_id = ?", sessionId).Order("created_at desc").Find(&chats)
	return chats, result.Error
}

func (r *ChatRepository) ListChatsUser(ctx context.Context, uh string) ([]*model.Chat, error) {
	var chats []*model.Chat
	res := r.db.WithContext(ctx).Where("user_hash = ?", uh).Order("created_at desc").Find(&chats)
	return chats, res.Error
}

func (r *ChatRepository) DeleteChat(ctx context.Context, id uuid.UUID) error {
	result := r.db.Delete(&model.Chat{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
