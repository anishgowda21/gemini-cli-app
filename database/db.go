package database

import (
	"fmt"
	"time"

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


func InitDB() error{
	var err error;

	db,err = gorm.Open(sqlite.Open("convo.db"), &gorm.Config{})

	if err != nil {
		return fmt.Errorf("Failed to connect DB: %w",err)
	}

	if err:= db.AutoMigrate(&Conversation{},&Message{}); err != nil{
		return fmt.Errorf("Failed to auto-migrate: %w", err)
	}

	return nil;
}
