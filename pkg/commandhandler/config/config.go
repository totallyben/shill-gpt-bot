package config

import (
	"time"

	"gitlab.totallydev.com/gritzb/shill-gpt-bot/pkg/storage"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Config struct {
	ConfigRepository `json:"-" bson:"-"`
	ID               primitive.ObjectID `bson:"_id,omitempty"`
	ChatID           int64              `bson:"chatId"`
	Token            string             `bson:"token"`
	Community        string             `bson:"community"`
	Hashtags         string             `bson:"hashtags"`
	Cashtags         string             `bson:"cashtags"`
	Created          time.Time
	Updated          time.Time
}

// NewShill
func NewConfig(mongo *storage.Mongo) Config {
	return Config{
		ConfigRepository: NewConfigRepository(mongo),
	}
}

// ConfigByChatID
func ConfigByChatID(mongo *storage.Mongo, ChatID int64) (Config, bool, error) {
	sl := NewConfig(mongo)

	filter := bson.D{
		{"chatId", ChatID},
	}

	results, err := sl.Find(filter, options.Find())

	if err != nil {
		return sl, false, err
	}

	if len(results) == 0 {
		return sl, false, nil
	}

	sl = results[0]
	sl.ConfigRepository = NewConfigRepository(mongo)

	return sl, true, nil
}
