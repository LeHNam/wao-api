package context

import (
	"context"
	"github.com/LeHNam/wao-api/models"
	"github.com/LeHNam/wao-api/services/database"
	"log"
	"time"

	"github.com/LeHNam/wao-api/config"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ServiceContext struct {
	Config *config.Config
	DB     *gorm.DB

	ctx               context.Context
	cancel            context.CancelFunc
	Log               *zap.Logger
	shutdown          chan struct{}
	ProductRepo       database.Repository[models.Product]
	ProductOptionRepo database.Repository[models.ProductOption]
	UserRepo          database.Repository[models.User]
}

func NewServiceContext(cfg *config.Config, db *gorm.DB, log *zap.Logger) *ServiceContext {
	ctx, cancel := context.WithCancel(context.Background())
	return &ServiceContext{
		Config: cfg,
		DB:     db,

		ctx:               ctx,
		cancel:            cancel,
		Log:               log,
		shutdown:          make(chan struct{}),
		ProductRepo:       models.NewProduct(db),
		ProductOptionRepo: models.NewProductOption(db),
		UserRepo:          models.NewUser(db),
	}
}

func (sc *ServiceContext) Context() context.Context {
	return sc.ctx
}

func (sc *ServiceContext) Shutdown() {
	sc.cancel()

	// gracefully shutdown the database connection
	sqlDB, err := sc.DB.DB()
	if err == nil {
		// wait for the database connection to be closed
		_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		sqlDB.Close()
	}

	close(sc.shutdown)
}

func (s *ServiceContext) Wait() {
	<-s.shutdown
	log.Println("ServiceContext shutdown complete")
}
