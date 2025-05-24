//go:build wireinject
// +build wireinject

package wire

import (
	"github.com/LeHNam/wao-api/config"
	"github.com/LeHNam/wao-api/services/database"
	"github.com/LeHNam/wao-api/services/log"
	"github.com/LeHNam/wao-api/services/server"
	"github.com/google/wire"
)

func InitializeServer() (*server.Server, error) {
	wire.Build(
		config.LoadConfig,
		log.NewZapLogger,
		server.NewServer,
		server.NewServiceContext,
		database.NewPostgresConnection,
	)

	return &server.Server{}, nil
}
