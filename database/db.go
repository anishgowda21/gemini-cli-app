package database

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Conversation struct {
	ID        string    `gorm:"primaryKey"`
	Title     string    `gorm:"not null"`
	Model     string    `gorm:"not null"`
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP"`
	Messages  []Message `gorm:"foreignKey:ConversationID"`
}

type Message struct {
	ID             string    `gorm:"primaryKey"`
	ConversationID string    `gorm:"not null;index"`
	Role           string    `gorm:"not null"` // "user" or "assistant"
	Content        string    `gorm:"not null"` // full message text
	RawContent     string    `gorm:"not null"` // message without extra formatting
	Thinking       *string   // optional
	ThinkingTime   *float64  // optional, time spent thinking (seconds)
	CreatedAt      time.Time `gorm:"default:CURRENT_TIMESTAMP"`
}

var db *gorm.DB

func InitDB() error {
	var err error

	db, err = gorm.Open(sqlite.Open("convo.db"), &gorm.Config{})

	if err != nil {
		return fmt.Errorf("Failed to connect DB: %w", err)
	}

	if err := db.AutoMigrate(&Conversation{}, &Message{}); err != nil {
		return fmt.Errorf("Failed to auto-migrate: %w", err)
	}

	return nil
}

func CreateConversation(title, model string) (string, error) {
	convoID := uuid.New().String()
	convo := Conversation{
		ID:        convoID,
		Title:     title,
		Model:     model,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	result := db.Create(&convo)

	if result.Error != nil {
		return "", fmt.Errorf("error creating conversation: %w", result.Error)
	}

	return convoID, nil
}

func AddMessage(convoID, role, content string) error {
	message := Message{
		ID:             uuid.New().String(),
		ConversationID: convoID,
		Role:           role,
		Content:        content,
		RawContent:     content,
		CreatedAt:      time.Now(),
	}

	result := db.Create(&message)

	if result.Error != nil {
		return fmt.Errorf("error adding message: %w", result.Error)
	}

	return nil
}

func GetMessagesByConversationID(convoID string) ([]Message, error) {
	var messages []Message

	result := db.Where("conversation_id = ?", convoID).Order("created_at asc").Find(&messages)

	if result.Error != nil {
		return nil, fmt.Errorf("error retrieving messages: %w", result.Error)
	}

	return messages, nil
}

func GetConversationByID(convoID string) (*Conversation, error) {
	var convo Conversation

	result := db.Preload("Messages").First(&convo, "id = ?", convoID)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("error retrieving conversation: %w", result.Error)
	}

	return &convo, nil
}

func ListConversations() ([]Conversation, error) {
	var convos []Conversation

	result := db.Find(&convos)

	if result.Error != nil {
		return nil, fmt.Errorf("error retrieving conversations: %w", result.Error)
	}
	return convos, nil
}

func UpdateConversation(convoID string) error {
	result := db.Model(&Conversation{}).Where("id = ?", convoID).Update("updated_at", time.Now())

	if result.Error != nil {
		return fmt.Errorf("error updating conversations: %w", result.Error)
	}

	return nil
}

func DeleteConversation(convoID string) error {

	if err := db.Where("conversation_id = ?", convoID).Delete(&Message{}).Error; err != nil {
		return fmt.Errorf("error deleting messages: %w", err)
	}

	if err := db.Delete(&Conversation{}, "id = ?", convoID).Error; err != nil {
		return fmt.Errorf("error deleting conversation: %w", err)
	}

	return nil
}
