package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Mongo - mongo client
type Mongo struct {
	*mongo.Client
}

// Collection - return mongo collection to work with
func (m *Mongo) Collection(collection string) *mongo.Collection {
	return m.Database(viper.GetString("mongo.DB")).Collection(collection)
}

// NewMongo - create a new mongo instance
func NewMongo() *Mongo {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dsn := fmt.Sprintf("mongodb://%s:%s", viper.GetString("mongo.host"), viper.GetString("mongo.port"))
	credential := options.Credential{
		AuthSource: viper.GetString("mongo.authSource"),
		Username:   viper.GetString("mongo.username"),
		Password:   viper.GetString("mongo.password"),
	}
	clientOpts := options.Client().ApplyURI(dsn).SetAuth(credential)

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		panic(err)
	}

	return &Mongo{client}
}
