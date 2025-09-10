package containers

import (
	"context"
	"fmt"

	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/queue/consumer"
	"gitlab.com/interview-simulation/interview-backend-server/internal/services"
)

func (c *Container) ServiceProvider() {
	if err := c.Container.Provide(services.NewUserService); err != nil {
		c.Error = err
	}

	if err := c.Container.Provide(services.NewResumeService); err != nil {
		c.Error = err
	}

	if err := c.Container.Provide(services.NewInterviewSessionService); err != nil {
		c.Error = err
	}

	if err := c.Container.Provide(services.NewEvaluationService); err != nil {
		c.Error = err
	}

	// Register consumer after services are available
	if err := c.Container.Provide(consumer.NewRedisTaskConsumer); err != nil {
		c.Error = err
	}

	// Start the consumer
	if err := c.Container.Invoke(func(consumer consumer.RedisTaskConsumer) {
		if err := consumer.CleanupQueue(context.Background()); err != nil {
			panic(fmt.Sprintf("Failed to cleanup queue: %v", err))
		}

		if err := consumer.Start(context.Background()); err != nil {
			panic(err)
		}
	}); err != nil {
		panic(err)
	}
}
