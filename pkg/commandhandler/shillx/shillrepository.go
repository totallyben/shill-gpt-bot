package shillx

import (
	"context"
	"time"

	"gitlab.totallydev.com/gritzb/shill-gpt-bot/pkg/storage"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ShillRepository interface {
	Insert(s *Shill) error
	Find(filter bson.D, findOptions *options.FindOptions) ([]Shill, error)
	Collection() *mongo.Collection
}

// NewShillRepository
func NewShillRepository(mongo *storage.Mongo) ShillRepository {
	return &shillRepository{mongo: mongo}
}

type shillRepository struct {
	mongo *storage.Mongo
}

// Insert
func (sr *shillRepository) Insert(s *Shill) error {
	s.Created = time.Now()

	result, err := sr.Collection().InsertOne(
		context.Background(),
		s,
	)

	if err != nil {
		return err
	}

	s.ID = result.InsertedID.(primitive.ObjectID)

	return err
}

// Find
func (sr *shillRepository) Find(filter bson.D, findOptions *options.FindOptions) ([]Shill, error) {
	var shills []Shill

	ctx := context.Background()
	cur, err := sr.Collection().Find(ctx, filter, findOptions)
	if err != nil {
		return shills, err
	}

	err = cur.All(ctx, &shills)

	return shills, err
}

// Collection
func (sr *shillRepository) Collection() *mongo.Collection {
	return sr.mongo.Collection("shill")
}
