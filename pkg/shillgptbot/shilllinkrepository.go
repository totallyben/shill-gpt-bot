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

type ShillLinkRepository interface {
	Insert(sl *ShillLink) error
	FindByID(ID primitive.ObjectID) *ShillLink
	Find(filter bson.D, findOptions *options.FindOptions) ([]ShillLink, error)
	Collection() *mongo.Collection
}

// NewShillLinkRepository
func NewShillLinkRepository(mongo *storage.Mongo) ShillLinkRepository {
	return &shillLinkRepository{mongo: mongo}
}

type shillLinkRepository struct {
	mongo *storage.Mongo
}

// Insert
func (slr *shillLinkRepository) Insert(sl *ShillLink) error {
	sl.Created = time.Now()

	result, err := slr.Collection().InsertOne(
		context.Background(),
		sl,
	)

	if err != nil {
		return err
	}

	sl.ID = result.InsertedID.(primitive.ObjectID)

	return err
}

// FindByID
func (slr *shillLinkRepository) FindByID(ID primitive.ObjectID) *ShillLink {
	sl := NewShillLink(slr.mongo)

	slr.Collection().FindOne(
		context.Background(),
		bson.M{"_id": ID},
	).Decode(sl)

	return sl
}

// Find
func (slr *shillLinkRepository) Find(filter bson.D, findOptions *options.FindOptions) ([]ShillLink, error) {
	var shillLinks []ShillLink

	ctx := context.Background()
	cur, err := slr.Collection().Find(ctx, filter, findOptions)
	if err != nil {
		return shillLinks, err
	}

	err = cur.All(ctx, &shillLinks)

	return shillLinks, err
}

// Collection
func (slr *shillLinkRepository) Collection() *mongo.Collection {
	return slr.mongo.Collection("shillLink")
}
