package models

import "go.mongodb.org/mongo-driver/v2/bson"

type PollOption struct {
	ID    string `json:"id" bson:"id"`
	Label string `json:"label" bson:"label"`
	Text  string `json:"text,omitempty" bson:"text,omitempty"`
	Votes int    `json:"votes" bson:"votes"`
}

type Poll struct {
	ID         bson.ObjectID `json:"id" bson:"_id,omitempty"`
	Question   string        `json:"question" bson:"question"`
	Options    []PollOption  `json:"options" bson:"options"`
	Status     string        `json:"status" bson:"status"`
	TotalVotes int           `json:"totalVotes" bson:"totalVotes"`
	CreatedBy  string        `json:"createdBy" bson:"createdBy"`
	CreatedAt  string        `json:"createdAt" bson:"createdAt"`
}

type Vote struct {
	ID        bson.ObjectID `json:"id" bson:"_id,omitempty"`
	PollID    string        `json:"pollId" bson:"pollId"`
	UserID    string        `json:"userId" bson:"userId"`
	OptionID  string        `json:"optionId" bson:"optionId"`
	CreatedAt string        `json:"createdAt" bson:"createdAt"`
}
