package routes

import (
	"github.com/gin-gonic/gin"

	"live-polling-backend/handlers"
)

func RegisterPollRoutes(router *gin.Engine) {
	router.GET("/polls/:id", handlers.GetPoll)
	router.GET("/polls/:id/stream", handlers.StreamPoll)

	protected := router.Group("/polls")
	protected.Use(handlers.RequireAuth)
	protected.POST("", handlers.CreatePoll)
	protected.GET("", handlers.GetPolls)
	protected.GET("/:id/my-vote", handlers.GetMyVote)
	protected.POST("/:id/vote", handlers.VotePoll)
}
