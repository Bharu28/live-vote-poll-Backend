package models

import "go.mongodb.org/mongo-driver/v2/bson"

type User struct {
	ID           bson.ObjectID `json:"id" bson:"_id,omitempty"`
	Name         string        `json:"name" bson:"name"`
	Email        string        `json:"email" bson:"email"`
	PasswordHash string        `json:"-" bson:"passwordHash"`
	CreatedAt    string        `json:"createdAt" bson:"createdAt"`
}
