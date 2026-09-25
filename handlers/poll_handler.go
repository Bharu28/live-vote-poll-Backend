package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"live-polling-backend/config"
	"live-polling-backend/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var errAlreadyVoted = errors.New("user has already voted in this poll")

func pollsCollection() *mongo.Collection {
	databaseName := os.Getenv("MONGODB_DATABASE")
	if databaseName == "" {
		databaseName = "pulsepoll"
	}

	return config.MongoClient.Database(databaseName).Collection("polls")
}

func votesCollection() *mongo.Collection {
	databaseName := os.Getenv("MONGODB_DATABASE")
	if databaseName == "" {
		databaseName = "pulsepoll"
	}

	return config.MongoClient.Database(databaseName).Collection("votes")
}

func isDuplicateVoteError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, errAlreadyVoted) {
		return true
	}

	var writeErr mongo.WriteException
	if errors.As(err, &writeErr) {
		for _, item := range writeErr.WriteErrors {
			if item.Code == 11000 {
				return true
			}
		}
	}

	return false
}

func currentUserIDFromJWT(c *gin.Context) (string, error) {
	authorization := strings.TrimSpace(c.GetHeader("Authorization"))
	if authorization == "" {
		return "", errors.New("authentication required")
	}

	parts := strings.SplitN(authorization, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errors.New("invalid authorization header")
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", errors.New("JWT_SECRET is not configured")
	}

	token, err := jwt.ParseWithClaims(parts[1], &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		if err == nil {
			err = errors.New("invalid token")
		}
		return "", err
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || claims.Subject == "" {
		return "", errors.New("invalid token claims")
	}

	return claims.Subject, nil
}

func RequireAuth(c *gin.Context) {
	if _, err := currentUserIDFromJWT(c); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Authentication required."})
		c.Abort()
		return
	}

	c.Next()
}

func getVoteForUserAndPoll(ctx context.Context, pollID, userID string) (models.Vote, bool, error) {
	var vote models.Vote
	if err := votesCollection().FindOne(ctx, bson.M{"pollId": pollID, "userId": userID}).Decode(&vote); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return models.Vote{}, false, nil
		}
		return models.Vote{}, false, err
	}

	return vote, true, nil
}

type CreatePollRequest struct {
	Question string   `json:"question"`
	Options  []string `json:"options"`
}

type VoteRequest struct {
	OptionID string `json:"optionId"`
}

func normalizePollOptions(values []string) []models.PollOption {
	normalized := make([]models.PollOption, 0, len(values))
	for i, value := range values {
		label := strings.TrimSpace(value)
		if label == "" {
			label = fmt.Sprintf("Option %d", i+1)
		}

		normalized = append(normalized, models.PollOption{
			ID:    bson.NewObjectID().Hex(),
			Label: label,
			Text:  label,
			Votes: 0,
		})
	}
	return normalized
}

func normalizePollOptionObjects[T any](items []T) []models.PollOption {
	normalized := make([]models.PollOption, 0, len(items))
	for i, item := range items {
		payload, err := json.Marshal(item)
		if err != nil {
			continue
		}
		var row struct {
			ID    string `json:"id"`
			Text  string `json:"text"`
			Label string `json:"label"`
			Value string `json:"value"`
		}
		if err := json.Unmarshal(payload, &row); err != nil {
			continue
		}

		label := strings.TrimSpace(row.Label)
		if label == "" {
			label = strings.TrimSpace(row.Text)
		}
		if label == "" {
			label = strings.TrimSpace(row.Value)
		}
		if label == "" {
			label = fmt.Sprintf("Option %d", i+1)
		}

		optionID := strings.TrimSpace(row.ID)
		if optionID == "" {
			optionID = bson.NewObjectID().Hex()
		}

		normalized = append(normalized, models.PollOption{
			ID:    optionID,
			Label: label,
			Text:  label,
			Votes: 0,
		})
	}
	return normalized
}

func parseCreatePollRequest(body []byte) (string, []models.PollOption, error) {
	var raw struct {
		Question string          `json:"question"`
		Options  json.RawMessage `json:"options"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return "", nil, err
	}

	question := strings.TrimSpace(raw.Question)
	if question == "" {
		return "", nil, fmt.Errorf("question is required")
	}

	if len(raw.Options) == 0 || strings.TrimSpace(string(raw.Options)) == "null" {
		return "", nil, fmt.Errorf("at least 2 options are required")
	}

	var stringOptions []string
	if err := json.Unmarshal(raw.Options, &stringOptions); err == nil && len(stringOptions) > 0 {
		return question, normalizePollOptions(stringOptions), nil
	}

	var objectOptions []struct {
		ID    string `json:"id"`
		Text  string `json:"text"`
		Label string `json:"label"`
		Value string `json:"value"`
	}
	if err := json.Unmarshal(raw.Options, &objectOptions); err == nil && len(objectOptions) > 0 {
		return question, normalizePollOptionObjects(objectOptions), nil
	}

	return "", nil, fmt.Errorf("options must be strings or objects")
}

func ensurePollDefaults(poll *models.Poll) {
	if poll == nil {
		return
	}
	if poll.Status == "" {
		poll.Status = "live"
	}
	poll.TotalVotes = 0
	for i := range poll.Options {
		if poll.Options[i].ID == "" {
			poll.Options[i].ID = bson.NewObjectID().Hex()
		}
		if poll.Options[i].Label == "" {
			poll.Options[i].Label = strings.TrimSpace(poll.Options[i].Text)
		}
		if poll.Options[i].Label == "" {
			poll.Options[i].Label = fmt.Sprintf("Option %d", i+1)
		}
		if poll.Options[i].Text == "" {
			poll.Options[i].Text = poll.Options[i].Label
		}
		poll.TotalVotes += poll.Options[i].Votes
	}
}

func getPollByID(pollID string) (models.Poll, error) {
	objectID, err := bson.ObjectIDFromHex(pollID)
	if err != nil {
		return models.Poll{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var poll models.Poll
	if err := pollsCollection().FindOne(ctx, bson.M{"_id": objectID}).Decode(&poll); err != nil {
		return models.Poll{}, err
	}

	ensurePollDefaults(&poll)
	return poll, nil
}

func publishPollUpdate(poll models.Poll) {
	if config.RedisClient == nil {
		return
	}

	payload, err := json.Marshal(poll)
	if err != nil {
		log.Println("failed to marshal poll update:", err)
		return
	}

	if err := config.RedisClient.Publish(context.Background(), pollUpdateChannel(poll.ID.Hex()), payload).Err(); err != nil {
		log.Println("failed to publish poll update:", err)
	}
}

func pollUpdateChannel(pollID string) string {
	return "poll:" + pollID
}

func CreatePoll(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Request body must be valid JSON."})
		return
	}

	question, options, err := parseCreatePollRequest(body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(options) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least 2 options are required."})
		return
	}

	poll := models.Poll{
		ID:        bson.NewObjectID(),
		Question:  question,
		Options:   options,
		Status:    "live",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	ensurePollDefaults(&poll)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	if _, err = pollsCollection().InsertOne(ctx, poll); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create poll."})
		return
	}

	publishPollUpdate(poll)
	c.JSON(http.StatusCreated, poll)
}

func GetPolls(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	cursor, err := pollsCollection().Find(ctx, bson.D{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch polls."})
		return
	}
	defer cursor.Close(ctx)

	var polls []models.Poll
	if err := cursor.All(ctx, &polls); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read polls."})
		return
	}
	if polls == nil {
		polls = []models.Poll{}
	}
	for i := range polls {
		ensurePollDefaults(&polls[i])
	}

	c.JSON(http.StatusOK, polls)
}

func GetPoll(c *gin.Context) {
	poll, err := getPollByID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Poll not found."})
		return
	}
	c.JSON(http.StatusOK, poll)
}

func GetMyVote(c *gin.Context) {
	userID, err := currentUserIDFromJWT(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Authentication required."})
		return
	}

	pollID := c.Param("id")
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	vote, exists, err := getVoteForUserAndPoll(ctx, pollID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Could not check your vote status."})
		return
	}

	if !exists {
		c.JSON(http.StatusOK, gin.H{"hasVoted": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{"hasVoted": true, "optionId": vote.OptionID})
}

func VotePoll(c *gin.Context) {
	pollID := c.Param("id")
	userID, err := currentUserIDFromJWT(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Authentication required."})
		return
	}

	var input VoteRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Request body must be valid JSON."})
		return
	}

	poll, err := getPollByID(pollID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Poll not found."})
		return
	}

	if poll.Status != "live" {
		c.JSON(http.StatusConflict, gin.H{"message": "This poll is no longer accepting votes."})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	if _, exists, err := getVoteForUserAndPoll(ctx, pollID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Could not check vote status."})
		return
	} else if exists {
		c.JSON(http.StatusConflict, gin.H{"message": "You have already voted in this poll."})
		return
	}

	optionIndex := -1
	for i := range poll.Options {
		if poll.Options[i].ID == input.OptionID {
			optionIndex = i
			break
		}
	}
	if optionIndex == -1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Selected option was not found."})
		return
	}

	voteDoc := models.Vote{
		ID:        bson.NewObjectID(),
		PollID:    pollID,
		UserID:    userID,
		OptionID:  input.OptionID,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	if _, err = votesCollection().InsertOne(ctx, voteDoc); err != nil {
		if isDuplicateVoteError(err) {
			c.JSON(http.StatusConflict, gin.H{"message": "You have already voted in this poll."})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not record vote."})
		return
	}

	poll.Options[optionIndex].Votes++
	ensurePollDefaults(&poll)

	objectID, err := bson.ObjectIDFromHex(pollID)
	if err != nil {
		if _, rollbackErr := votesCollection().DeleteOne(ctx, bson.M{"_id": voteDoc.ID}); rollbackErr != nil {
			log.Println("failed to rollback vote after invalid poll ID:", rollbackErr)
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid poll ID."})
		return
	}

	if _, err = pollsCollection().ReplaceOne(ctx, bson.M{"_id": objectID}, poll); err != nil {
		if _, rollbackErr := votesCollection().DeleteOne(ctx, bson.M{"_id": voteDoc.ID}); rollbackErr != nil {
			log.Println("failed to rollback vote after poll update error:", rollbackErr)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save vote."})
		return
	}

	publishPollUpdate(poll)
	c.JSON(http.StatusOK, poll)
}

func StreamPoll(c *gin.Context) {
	pollID := c.Param("id")
	if _, err := getPollByID(pollID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Poll not found."})
		return
	}

	if config.RedisClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Realtime streaming is unavailable."})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache, no-transform")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Flush()

	currentPoll, _ := getPollByID(pollID)
	payload, _ := json.Marshal(currentPoll)
	if _, err := c.Writer.WriteString("data: " + string(payload) + "\n\n"); err == nil {
		c.Writer.Flush()
	}

	pubsub := config.RedisClient.Subscribe(context.Background(), pollUpdateChannel(pollID))
	defer pubsub.Close()

	channel := pubsub.Channel()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case msg, ok := <-channel:
			if !ok {
				return
			}
			if _, err := c.Writer.WriteString("data: " + msg.Payload + "\n\n"); err == nil {
				c.Writer.Flush()
			}
		case <-time.After(30 * time.Second):
			if _, err := c.Writer.WriteString("data: {\"status\":\"ping\"}\n\n"); err == nil {
				c.Writer.Flush()
			}
		}
	}
}
