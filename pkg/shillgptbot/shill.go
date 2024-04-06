package shillgptbot

import (
	"time"

	"gitlab.totallydev.com/gritzb/shill-gpt-bot/pkg/storage"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Shill struct {
	ShillRepository `json:"-" bson:"-"`
	ID              primitive.ObjectID `bson:"_id,omitempty"`
	ChatID          int64              `bson:"chatId"`
	TweetID         string             `bson:"tweetId"`
	Reply           string             `bson:"reply"`
	Created         time.Time
}

// NewShill
func NewShill(mongo *storage.Mongo) *Shill {
	return &Shill{
		ShillRepository: NewShillRepository(mongo),
	}
}
