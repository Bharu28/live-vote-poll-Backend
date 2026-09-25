package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"live-polling-backend/config"
	"live-polling-backend/handlers"
	"live-polling-backend/models"
	"live-polling-backend/routes"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

type authRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string      `json:"token"`
	User  models.User `json:"user"`
}

func signupHandler(c *gin.Context) {
	var input authRequest

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Request body must be valid JSON.",
		})
		return
	}

	name := strings.TrimSpace(input.Name)
	email := strings.ToLower(strings.TrimSpace(input.Email))
	password := input.Password

	if len(name) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Name is required.",
		})
		return
	}

	if !validEmail(email) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Please enter a valid email address.",
		})
		return
	}

	if len(password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Password must be at least 6 characters.",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var existingUser models.User

	err := usersCollection().FindOne(
		ctx,
		bson.D{{Key: "email", Value: email}},
	).Decode(&existingUser)

	if err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "An account with this email already exists.",
		})
		return
	}

	if err != mongo.ErrNoDocuments {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Could not check existing account.",
		})
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Could not secure password.",
		})
		return
	}

	user := models.User{
		ID:           bson.NewObjectID(),
		Name:         name,
		Email:        email,
		PasswordHash: string(passwordHash),
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
	}

	_, err = usersCollection().InsertOne(ctx, user)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Could not create account.",
		})
		return
	}

	token, err := issueToken(user)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Could not issue authentication.",
		})
		return
	}

	c.JSON(http.StatusCreated, authResponse{
		Token: token,
		User:  user,
	})
}

func loginHandler(c *gin.Context) {
	var input authRequest

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Request body must be valid JSON.",
		})
		return
	}

	email := strings.ToLower(strings.TrimSpace(input.Email))

	ctx, cancel := context.WithTimeout(
		c.Request.Context(),
		10*time.Second,
	)
	defer cancel()

	var user models.User

	if err := usersCollection().
		FindOne(ctx, bson.D{{Key: "email", Value: email}}).
		Decode(&user); err != nil {

		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Email or password is incorrect.",
		})
		return
	}

	if bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(input.Password),
	) != nil {

		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Email or password is incorrect.",
		})
		return
	}

	token, err := issueToken(user)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Could not issue authentication.",
		})
		return
	}

	c.JSON(http.StatusOK, authResponse{
		Token: token,
		User:  user,
	})
}

func usersCollection() *mongo.Collection {
	databaseName := os.Getenv("MONGODB_DATABASE")

	if databaseName == "" {
		databaseName = "pulsepoll"
	}

	return config.MongoClient.
		Database(databaseName).
		Collection("users")
}

func createUserIndex() {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	_, err := usersCollection().Indexes().CreateOne(
		ctx,
		mongo.IndexModel{
			Keys: bson.D{
				{Key: "email", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
	)

	if err != nil {
		panic(err)
	}
}

func createVoteIndex() {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	databaseName := os.Getenv("MONGODB_DATABASE")
	if databaseName == "" {
		databaseName = "pulsepoll"
	}

	_, err := config.MongoClient.
		Database(databaseName).
		Collection("votes").Indexes().CreateOne(
		ctx,
		mongo.IndexModel{
			Keys: bson.D{
				{Key: "pollId", Value: 1},
				{Key: "userId", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
	)

	if err != nil {
		panic(err)
	}
}

func issueToken(user models.User) (string, error) {
	secret := os.Getenv("JWT_SECRET")

	if secret == "" {
		return "", errors.New("JWT_SECRET is not configured")
	}

	claims := jwt.RegisteredClaims{
		Subject:   user.ID.Hex(),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	return token.SignedString([]byte(secret))
}

func validEmail(email string) bool {
	at := strings.LastIndex(email, "@")

	return at > 0 &&
		at < len(email)-3 &&
		strings.Contains(email[at+1:], ".")
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		requestOrigin := c.GetHeader("Origin")

		configuredOrigins := []string{
			"http://localhost:5173",
			"http://127.0.0.1:5173",
		}

		if envOrigins := strings.TrimSpace(
			os.Getenv("FRONTEND_ORIGIN"),
		); envOrigins != "" {

			configuredOrigins = nil

			for _, value := range strings.Split(
				envOrigins,
				",",
			) {
				trimmed := strings.TrimSpace(value)

				if trimmed != "" {
					configuredOrigins = append(
						configuredOrigins,
						trimmed,
					)
				}
			}
		}

		isAllowedOrigin := func(origin string) bool {

			if origin == "" {
				return false
			}

			for _, allowedOrigin := range configuredOrigins {
				if origin == allowedOrigin {
					return true
				}
			}

			return strings.HasPrefix(
				origin,
				"http://localhost:",
			) ||
				strings.HasPrefix(
					origin,
					"http://127.0.0.1:",
				) ||
				strings.HasPrefix(
					origin,
					"http://[::1]:",
				)
		}

		if isAllowedOrigin(requestOrigin) {
			c.Header(
				"Access-Control-Allow-Origin",
				requestOrigin,
			)
		} else if requestOrigin == "" &&
			len(configuredOrigins) > 0 {

			c.Header(
				"Access-Control-Allow-Origin",
				configuredOrigins[0],
			)
		}

		c.Header("Vary", "Origin")

		c.Header(
			"Access-Control-Allow-Headers",
			"Content-Type, Authorization",
		)

		c.Header(
			"Access-Control-Allow-Methods",
			"GET, POST, OPTIONS",
		)

		if c.Request.Method == http.MethodOptions {
			c.Status(http.StatusNoContent)
			c.Abort()
			return
		}

		c.Next()
	}
}

func main() {

	config.ConnectDatabase()

	createUserIndex()
	createVoteIndex()

	router := gin.Default()

	router.Use(corsMiddleware())

	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Live Polling API is running",
		})
	})

	// Authentication
	router.POST("/auth/signup", signupHandler)
	router.POST("/auth/login", loginHandler)

	// Poll routes
	routes.RegisterPollRoutes(router)

	_ = handlers.CreatePoll

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := router.Run(":" + port); err != nil {
		panic(err)
	}
}
