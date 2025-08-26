package containers

import (
	"context"
	"database/sql"
	"fmt"

	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	db "gitlab.com/interview-simulation/interview-backend-server/internal/db/sqlc"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/aws"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/database"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/email"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/queue"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/server"
	ws "gitlab.com/interview-simulation/interview-backend-server/internal/infras/websocket"
	"gitlab.com/interview-simulation/interview-backend-server/internal/middleware"
	"gitlab.com/interview-simulation/interview-backend-server/internal/utils"
	"gorm.io/gorm"
)

func (c *Container) InfrastructureProvider() {

	c.Container.Provide(func(cfg *config.Config) *log.Logger {
		return log.Initialize(cfg.AppConfig.AppEnv)
	})

	if err := c.Container.Provide(func(cfg *config.Config) *server.GinServer {
		return server.NewGinServer(cfg, c.Container)
	}); err != nil {
		c.Error = err
	}

	if err := c.Container.Provide(utils.NewJwtToken); err != nil {
		c.Error = err
	}

	if err := c.Container.Provide(utils.NewSignInTokenPayload); err != nil {
		c.Error = err
	}

	if err := c.Container.Provide(utils.NewVerifyEmailTokenPayload); err != nil {
		c.Error = err
	}

	if err := c.Container.Provide(utils.NewPassword); err != nil {
		c.Error = err
	}

	if err := c.Container.Provide(database.ConnectPostgres); err != nil {
		c.Error = err
	}

	if err := c.Container.Provide(func(conn *database.DBConnections) *gorm.DB {
		return conn.GormDB
	}); err != nil {
		c.Error = err
	}

	if err := c.Container.Provide(func(conn *database.DBConnections) *sql.DB {
		return conn.SqlDB
	}); err != nil {
		c.Error = err
	}

	if err := c.Container.Provide(database.NewRedisClient); err != nil {
		c.Error = err
	}

	if err := c.Container.Provide(db.NewStore); err != nil {
		c.Error = err
	}

	if err := c.Container.Provide(middleware.NewAuthMiddleware); err != nil {
		c.Error = err
	}

	if err := c.Container.Provide(middleware.NewAuthContext); err != nil {
		c.Error = err
	}

	if err := c.Container.Provide(email.NewEmailSender); err != nil {
		c.Error = err
	}

	if err := c.Container.Provide(queue.NewRedisTaskPublisher); err != nil {
		c.Error = err
	}

	if err := c.Container.Provide(aws.NewS3Storage); err != nil {
		c.Error = err
	}

	if err := c.Container.Provide(queue.NewRedisTaskConsumer); err != nil {
		c.Error = err
	}

	if err := c.Container.Invoke(func(consumer queue.RedisTaskConsumer) {
		if err := consumer.CleanupQueue(context.Background()); err != nil {
			panic(fmt.Sprintf("Failed to cleanup queue: %v", err))
		}

		if err := consumer.Start(context.Background()); err != nil {
			panic(err)
		}
	}); err != nil {
		panic(err)
	}

	if err := c.Container.Provide(utils.NewGenerator); err != nil {
		c.Error = err
	}

	if err := c.Container.Provide(utils.NewValidator); err != nil {
		c.Error = err
	}

	if err := c.Container.Provide(utils.NewCookies); err != nil {
		c.Error = err
	}

	if err := c.Container.Provide(ws.NewWebSocketServer); err != nil {
		c.Error = err
	}

	if err := c.Container.Provide(ws.NewWebSocketClient); err != nil {
		c.Error = err
	}
}
