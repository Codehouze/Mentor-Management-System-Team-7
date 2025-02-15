// Package api (task) defines functions to create and get Tasks from DB.
package api

import (
	"net/http"
	"time"

	"github.com/ALCOpenSource/Mentor-Management-System-Team-7/backend/db/models"
	"github.com/ALCOpenSource/Mentor-Management-System-Team-7/backend/internal/token"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type createTaskRequest struct {
	Title          string   `json:"title" binding:"required"`
	Details        string   `json:"details" binding:"required"`
	MentorManagers []string `json:"mentor_managers" binding:"required,min=1"`
	Mentors        []string `json:"mentors" binding:"required,min=1"`
}

func (server *Server) createTask(ctx *gin.Context) {
	var req createTaskRequest

	if err := bindJSONWithValidation(ctx, ctx.ShouldBindJSON(&req), validator.New()); err != nil {
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	if authPayload.UserRole != "Admin" && authPayload.UserRole != "Mentor Manager (MM)" {
		ctx.JSON(http.StatusUnauthorized, errorResponse("not authorised to create task"))
		return
	}

	// check if mentor managers exist in the database and get their ids
	mentorManagerIDs := []primitive.ObjectID{}
	for _, mentorManager := range req.MentorManagers {
		user, err := server.store.GetUserByID(ctx, mentorManager)
		if err != nil || user.Role != "Mentor Manager (MM)" {
			ctx.JSON(http.StatusInternalServerError, errorResponse("failed to get mentor manager"))
			return
		}

		mentorManagerIDs = append(mentorManagerIDs, user.ID)
	}

	// check if mentors exist in the database and get their ids
	mentorIDs := []primitive.ObjectID{}
	for _, mentor := range req.Mentors {
		user, err := server.store.GetUserByID(ctx, mentor)
		if err != nil || user.Role != "Mentor" {
			ctx.JSON(http.StatusInternalServerError, errorResponse("failed to get mentor"))
			return
		}
		mentorIDs = append(mentorIDs, user.ID)
	}

	task := &models.Task{
		Title:          req.Title,
		Details:        req.Details,
		MentorManagers: mentorManagerIDs,
		Mentors:        mentorIDs,
		CreatedAt:      time.Now(),
	}

	resp, err := server.store.CreateTask(ctx, task)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse("failed to create task"))
		return
	}

	ctx.JSON(http.StatusCreated, envelop{"data": resp})
	log.Info().
		Str("user_id", authPayload.UserID).
		Str("ip_address", ctx.ClientIP()).
		Str("user_agent", ctx.Request.UserAgent()).
		Str("request_method", ctx.Request.Method).
		Str("request_path", ctx.Request.URL.Path).
		Msg("task created")
}

// listTasks returns a list of tasks.
func (server *Server) listTasks(ctx *gin.Context) {
	tasks, err := server.store.ListTasks(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse("failed to list tasks"))
		return
	}
	// Todo: add mentor managers and mentors to the response
	ctx.JSON(http.StatusOK, envelop{"data": tasks})

}

// getTask returns a task by id.
func (server *Server) getTask(ctx *gin.Context) {
	taskID := ctx.Param("id")

	task, err := server.store.GetTask(ctx, taskID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse("failed to get task"))
		return
	}

	ctx.JSON(http.StatusOK, envelop{"data": task})
}

type updateTaskRequest struct {
	Title          string   `json:"title" binding:"required"`
	Details        string   `json:"details" binding:"required"`
	MentorManagers []string `json:"mentor_managers" binding:"required,min=1"`
	Mentors        []string `json:"mentors" binding:"required,min=1"`
}

// updateTask updates an existing task document in the collection.
func (server *Server) updateTask(ctx *gin.Context) {

	var req updateTaskRequest

	if err := bindJSONWithValidation(ctx, ctx.ShouldBindJSON(&req), validator.New()); err != nil {
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	if authPayload.UserRole != "Admin" && authPayload.UserRole != "Mentor Manager (MM)" {
		ctx.JSON(http.StatusUnauthorized, errorResponse("not authorised to update task"))
		return
	}

	// check if mentor managers exist in the database and get their ids
	mentorManagerIDs := []primitive.ObjectID{}
	for _, mentorManager := range req.MentorManagers {
		user, err := server.store.GetUserByID(ctx, mentorManager)
		if err != nil || user.Role != "Mentor Manager (MM)" {
			ctx.JSON(http.StatusInternalServerError, errorResponse("failed to get mentor manager"))
			return
		}

		mentorManagerIDs = append(mentorManagerIDs, user.ID)
	}

	// check if mentors exist in the database and get their ids
	mentorIDs := []primitive.ObjectID{}
	for _, mentor := range req.Mentors {
		user, err := server.store.GetUserByID(ctx, mentor)
		if err != nil || user.Role != "Mentor" {
			ctx.JSON(http.StatusInternalServerError, errorResponse("failed to get mentor"))
			return
		}
		mentorIDs = append(mentorIDs, user.ID)
	}

	taskID := ctx.Param("id")

	task, err := server.store.GetTask(ctx, taskID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse("failed to get task"))
		return
	}

	task.Title = req.Title
	task.Details = req.Details
	task.MentorManagers = mentorManagerIDs
	task.Mentors = mentorIDs

	resp, err := server.store.UpdateTask(ctx, taskID, task)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse("failed to update task"))
		return
	}

	ctx.JSON(http.StatusOK, envelop{"data": resp})
	log.Info().
		Str("user_id", authPayload.UserID).
		Str("ip_address", ctx.ClientIP()).
		Str("user_agent", ctx.Request.UserAgent()).
		Str("request_method", ctx.Request.Method).
		Str("request_path", ctx.Request.URL.Path).
		Msg("task updated")
}

// deleteTask deletes a task by id.
func (server *Server) deleteTask(ctx *gin.Context) {
	taskID := ctx.Param("id")

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	if authPayload.UserRole != "Admin" && authPayload.UserRole != "Mentor Manager (MM)" {
		ctx.JSON(http.StatusUnauthorized, errorResponse("not authorised to delete task"))
		return
	}

	_, err := server.store.DeleteTask(ctx, taskID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse("failed to delete task"))
		return
	}

	ctx.JSON(http.StatusOK, envelop{"data": "task deleted"})
	log.Info().
		Str("user_id", authPayload.UserID).
		Str("ip_address", ctx.ClientIP()).
		Str("user_agent", ctx.Request.UserAgent()).
		Str("request_method", ctx.Request.Method).
		Str("request_path", ctx.Request.URL.Path).
		Msg("task deleted")
}
