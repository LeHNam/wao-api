package server

import (
	"context"
	"fmt"
	"github.com/LeHNam/wao-api/api/product"
	purchaseOrder "github.com/LeHNam/wao-api/api/purchase_order"
	"github.com/LeHNam/wao-api/api/user"
	"github.com/LeHNam/wao-api/config"
	"github.com/LeHNam/wao-api/middlewares"
	"github.com/LeHNam/wao-api/services/i18nService"
	"github.com/LeHNam/wao-api/services/websocket"
	"github.com/getkin/kin-openapi/openapi3filter"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	svCtx "github.com/LeHNam/wao-api/context"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gin-gonic/gin"
	oMiddleware "github.com/oapi-codegen/gin-middleware"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Server struct {
	router     *gin.Engine
	httpServer *http.Server
	sc         *svCtx.ServiceContext
	wsService  *websocket.WebSocketService
}

func NewServer(sc *svCtx.ServiceContext, wsService *websocket.WebSocketService) *Server {
	router := gin.Default()
	router.Use(middlewares.CORSMiddleware())

	return &Server{
		router:    router,
		sc:        sc,
		wsService: wsService,
	}
}

func (s *Server) AutoMigrate() {
	err := s.sc.DB.AutoMigrate(
	//&models.PurchaseOrderItem{},
	//&models.User{},
	)
	if err != nil {

		log.Fatalf("error auto migrating models: %v", err)
	}
}

func (s *Server) SetupRoutes() {
	s.router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Welcome to TRY API")
	})

	s.router.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	// serve swagger ui
	s.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/swagger.yaml")))
	s.router.StaticFile("/swagger.yaml", "bundled.yaml")

	// Load API specification directly from api.yaml instead of using GetSwagger()
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	swagger, err := loader.LoadFromFile("bundled.yaml")
	if err != nil {
		log.Fatalf("error loading swagger spec from file: %v", err)
	}

	// Validate the swagger document
	err = swagger.Validate(context.Background())
	if err != nil {
		log.Fatalf("error validating swagger spec: %v", err)
	}

	// register api group with swagger validator
	authMiddlewareFactory := middlewares.BearerAuthMiddleware()
	apiPrefix := "/api/v1"
	apiGroupV1 := s.router.Group(
		apiPrefix,
		oMiddleware.OapiRequestValidatorWithOptions(swagger, &oMiddleware.Options{
			ErrorHandler: func(c *gin.Context, err string, statusCode int) {
				c.JSON(statusCode, gin.H{
					"message": "Validation failed",
					"error":   err,
				})
			},
			SilenceServersWarning: true,
			Options: openapi3filter.Options{
				AuthenticationFunc: func(c context.Context, input *openapi3filter.AuthenticationInput) error {
					fmt.Println("Authentication input:", input)
					return authMiddlewareFactory(c, input)
				},
			},
		}))

	apiGroupV1.Use()
	{
		userServer := user.NewUserServer(s.sc)
		userHandler := user.NewStrictHandler(userServer, nil)
		user.RegisterHandlersWithOptions(apiGroupV1, userHandler, user.GinServerOptions{})

		productServer := product.NewProductServer(s.sc, s.wsService)
		productHandler := product.NewStrictHandler(productServer, nil)
		product.RegisterHandlersWithOptions(apiGroupV1, productHandler, product.GinServerOptions{})

		purchaseOrderServer := purchaseOrder.NewPurchaseOrderServer(s.sc, s.wsService)
		purchaseOrderHandler := purchaseOrder.NewStrictHandler(purchaseOrderServer, nil)
		purchaseOrder.RegisterHandlersWithOptions(apiGroupV1, purchaseOrderHandler, purchaseOrder.GinServerOptions{})
	}

	s.router.GET("/ws", func(c *gin.Context) {
		s.wsService.HandleWebSocket(c.Writer, c.Request)
	})

}

func (s *Server) Run() error {
	configConfig, err := config.LoadConfig()
	if err != nil {
		return err
	}

	s.httpServer = &http.Server{
		Addr:    ":" + configConfig.Server.Port,
		Handler: s.router,
	}

	i18nService.NewI18nService()

	// graceful shutdown
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		log.Println("Shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.httpServer.Shutdown(ctx); err != nil {
			log.Fatal("Server forced to shutdown:", err)
		}

		// shutdown service context
		s.sc.Shutdown()
	}()

	log.Printf("Server is running on %s", configConfig.Server.Port)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	// service context wait for shutdown
	s.sc.Wait()

	return nil
}
