package storage

import (
	"testing"

	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

// Mongo - mongo client
type MockMongo struct {
	*mtest.T
}

// NewMongo - create a new mongo instance
func NewMockMongo(t *testing.T) *MockMongo {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	return &MockMongo{
		mt,
	}
}
