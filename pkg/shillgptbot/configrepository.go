package shillgptbot

import (
	"context"
	"time"

	"gitlab.totallydev.com/gritzb/shill-gpt-bot/pkg/storage"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ConfigRepository interface {
	Insert(c *Config) error
	Find(filter bson.D, findOptions *options.FindOptions) ([]Config, error)
	Collection() *mongo.Collection
}

// NewConfigRepository
func NewConfigRepository(mongo *storage.Mongo) ConfigRepository {
	return &configRepository{mongo: mongo}
}

type configRepository struct {
	mongo *storage.Mongo
}

// Insert
func (cr *configRepository) Insert(c *Config) error {
	c.Created = time.Now()
	c.Updated = time.Now()

	result, err := cr.Collection().InsertOne(
		context.Background(),
		c,
	)

	if err != nil {
		return err
	}

	c.ID = result.InsertedID.(primitive.ObjectID)

	return err
}

// Find
func (cr *configRepository) Find(filter bson.D, findOptions *options.FindOptions) ([]Config, error) {
	var configs []Config

	ctx := context.Background()
	cur, err := cr.Collection().Find(ctx, filter, findOptions)
	if err != nil {
		return configs, err
	}

	err = cur.All(ctx, &configs)

	return configs, err
}

// Collection
func (cr *configRepository) Collection() *mongo.Collection {
	return cr.mongo.Collection("config")
}
