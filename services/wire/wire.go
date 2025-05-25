//go:build wireinject
// +build wireinject

package wire

import (
	"github.com/LeHNam/wao-api/config"
	"github.com/LeHNam/wao-api/context"
	"github.com/LeHNam/wao-api/services/database"
	"github.com/LeHNam/wao-api/services/log"
	"github.com/LeHNam/wao-api/services/server"
	"github.com/LeHNam/wao-api/services/websocket"
	"github.com/google/wire"
)

func InitializeServer() (*server.Server, error) {
	wire.Build(
		config.LoadConfig,
		log.NewZapLogger,
		websocket.NewWebSocketService,
		context.NewServiceContext,
		server.NewServer,
		database.NewPostgresConnection,
	)

	return &server.Server{}, nil
}
