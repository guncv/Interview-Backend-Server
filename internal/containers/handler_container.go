package containers

import (
	"gitlab.com/interview-simulation/interview-backend-server/internal/handlers"
)

func (c *Container) HandlerProvider() {
	if err := c.Container.Provide(handlers.NewUserHandler); err != nil {
		c.Error = err
	}

	if err := c.Container.Provide(handlers.NewResumeHandler); err != nil {
		c.Error = err
	}

	if err := c.Container.Provide(handlers.NewInterviewSessionHandler); err != nil {
		c.Error = err
	}

	if err := c.Container.Provide(handlers.NewIssueReportsHandler); err != nil {
		c.Error = err
	}

	if err := c.Container.Provide(handlers.NewReviewCommentHandler); err != nil {
		c.Error = err
	}

	if err := c.Container.Provide(handlers.NewEvaluationHandler); err != nil {
		c.Error = err
	}
}
