package containers

import (
	"gitlab.com/interview-simulation/interview-backend-server/internal/services"
)

func (c *Container) ServiceProvider() {
	if err := c.Container.Provide(services.NewUserService); err != nil {
		c.Error = err
	}

	if err := c.Container.Provide(services.NewResumeService); err != nil {
		c.Error = err
	}

	if err := c.Container.Provide(services.NewWebSocketService); err != nil {
		c.Error = err
	}

	if err := c.Container.Provide(services.NewInterviewSessionService); err != nil {
		c.Error = err
	}
}
