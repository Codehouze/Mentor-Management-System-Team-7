// Package api provides an HTTP server implementation for the Mentor-Management-System-Team-7
// backend application. It defines a Server struct that serves HTTP requests and sets up the routing.
// The package utilizes the following internal packages: db, token, utils, and worker.
package api

import (
	"fmt"
	"io/fs"
	"net/http"

	"github.com/ALCOpenSource/Mentor-Management-System-Team-7/backend/db"
	"github.com/ALCOpenSource/Mentor-Management-System-Team-7/backend/internal/cache"
	"github.com/ALCOpenSource/Mentor-Management-System-Team-7/backend/internal/token"
	"github.com/ALCOpenSource/Mentor-Management-System-Team-7/backend/internal/utils"
	"github.com/ALCOpenSource/Mentor-Management-System-Team-7/backend/internal/worker"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/gin-gonic/gin"
)

// A Server serves HTTP requests for the banking system
type Server struct {
	config          utils.Config
	store           db.Store
	router          *gin.Engine
	tokenMaker      token.Maker
	taskDistributor worker.TaskDistributor
	swaggerFiles    fs.FS
	googleConfig    *oauth2.Config
	cache           cache.Cache
}

// NewServer create a new HTTP server and setup routing.
func NewServer(
	config utils.Config,
	store db.Store,
	taskDistributor worker.TaskDistributor,
	swaggerFiles fs.FS,
	cache cache.Cache,
) (*Server, error) {
	tokenMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}

	googleConfig := &oauth2.Config{
		Endpoint:     google.Endpoint,
		RedirectURL:  config.GoogleRedirectURL,
		ClientID:     config.GoogleClientID,
		ClientSecret: config.GoogleClientSecret,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
	}

	server := &Server{
		config:          config,
		store:           store,
		tokenMaker:      tokenMaker,
		taskDistributor: taskDistributor,
		swaggerFiles:    swaggerFiles,
		googleConfig:    googleConfig,
		cache:           cache,
	}

	server.setupRouter()

	return server, nil
}

func (s *Server) setupRouter() {
	router := gin.Default()
	router.Use(s.enableCORS(), loggerMiddleware())

	router.POST("/api/v1/forgot_password", s.forgotPassword)
	router.PATCH("/api/v1/reset_password", s.resetPassword)
	router.POST("/api/v1/auth/login", s.login)
	router.GET("/api/v1/auth/google/login", gin.WrapF(s.googleLogin))
	router.GET("/api/v1/auth/google/callback", s.googleLoginCallback)

	fsysHandler := http.FileServer(http.FS(s.swaggerFiles))
	router.GET("/api/v1/swagger/*any", gin.WrapH(http.StripPrefix("/api/v1/swagger/", fsysHandler)))

	authRoutes := router.Group("/").Use(s.authMiddleware(s.tokenMaker))
	authRoutes.PATCH("/api/v1/users/:id/change_password", s.changeUserPassword)
	authRoutes.POST("/api/v1/faqs", s.createFAQ)
	authRoutes.GET("/api/v1/faqs", s.getAllFAQs)
	authRoutes.PATCH("/api/v1/users/:id", s.updateUser)
	authRoutes.POST("/api/v1/discussions", s.createDiscussion)
	authRoutes.POST("/api/v1/discussions/:id/add_comment", s.addComment)
	authRoutes.GET("/api/v1/discussions", s.listDiscussions)
	authRoutes.GET("/api/v1/discussions/:id", s.getDiscussion)
	authRoutes.PATCH("/api/v1/discussions/:id", s.updateDiscussion)

	authRoutes.POST("/api/v1/tasks", s.createTask)
	authRoutes.GET("/api/v1/tasks", s.listTasks)
	authRoutes.PATCH("/api/v1/tasks/:id", s.updateTask)
	authRoutes.GET("/api/v1/tasks/:id", s.getTask)
	authRoutes.DELETE("/api/v1/tasks/:id", s.deleteTask)

	authRoutes.GET("/api/v1/mentors", s.listMentors)
	authRoutes.GET("/api/v1/mentor_managers", s.listMentorManagers)

	authRoutes.POST("/api/v1/programs", s.createProgram)
	authRoutes.GET("/api/v1/programs", s.listPrograms)
	authRoutes.GET("/api/v1/programs/:id", s.getProgram)
	authRoutes.PATCH("/api/v1/programs/:id", s.updateProgram)
	authRoutes.DELETE("/api/v1/programs/:id", s.deleteProgram)

	authRoutes.POST("/api/v1/criteria", s.createCriteria)
	authRoutes.GET("/api/v1/criteria", s.listCriteria)
	authRoutes.GET("/api/v1/criteria/:id", s.getCriteria)
	authRoutes.PATCH("/api/v1/criteria/:id", s.updateCriteria)
	authRoutes.DELETE("/api/v1/criteria/:id", s.deleteCriteria)

	
	authRoutes.POST("/api/v1/auth/logout", s.logout)

	s.router = router
}

// Start run the HTTP server on a specific address.
func (s *Server) Start(address string) error {
	return s.router.Run(address)
}

func errorResponse(err string) gin.H {
	return gin.H{"error": err}
}

type envelop map[string]interface{}
