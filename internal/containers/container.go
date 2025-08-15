package containers

import (
	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/server"
	"go.uber.org/dig"
)

type Container struct {
	Container *dig.Container
	Error     error
}

func (c *Container) Configure() {
	c.Container = dig.New()

	c.Container.Provide(config.LoadConfig)

	c.InfrastructureProvider()
	c.RepositoryProvider()
	c.ServiceProvider()
	c.HandlerProvider()
}

func (c *Container) Run() *Container {
	if err := c.Container.Invoke(func(s *server.GinServer) {
		if err := s.Start(); err != nil {
			panic(err)
		}
	}); err != nil {
		panic(err)
	}

	return c
}

func NewContainer() *Container {
	c := &Container{}
	c.Configure()
	return c
}
