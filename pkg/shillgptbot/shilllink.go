package shillgptbot

import (
	"time"

	"gitlab.totallydev.com/gritzb/shill-gpt-bot/pkg/storage"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ShillLink struct {
	ShillLinkRepository `json:"-" bson:"-"`
	ID                  primitive.ObjectID `bson:"_id,omitempty"`
	ChatID              int64              `bson:"chatId"`
	TweetID             string             `bson:"tweetId"`
	TweetLink           string             `bson:"tweetLink"`
	TweetText           string             `bson:"tweetText"`
	Created             time.Time
}

// NewShill
func NewShillLink(mongo *storage.Mongo) *ShillLink {
	return &ShillLink{
		ShillLinkRepository: NewShillLinkRepository(mongo),
	}
}

// ShillLinkByID
func ShillLinkByID(mongo *storage.Mongo, shillID string) (*ShillLink, bool, error) {
	sl := NewShillLink(mongo)

	objectID, err := primitive.ObjectIDFromHex(shillID)
	if err != nil {
		return sl, false, err
	}

	filter := bson.D{
		{"_id", objectID},
	}

	results, err := sl.Find(filter, options.Find())

	if err != nil {
		return sl, false, err
	}

	if len(results) == 0 {
		return sl, false, nil
	}

	sl = &results[0]
	sl.ShillLinkRepository = NewShillLinkRepository(mongo)

	return sl, true, nil
}
