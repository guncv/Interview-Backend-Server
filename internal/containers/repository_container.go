package containers

import (
	"gitlab.com/interview-simulation/interview-backend-server/internal/repositories"
)

func (c *Container) RepositoryProvider() {
	if err := c.Container.Provide(repositories.NewUserRepository); err != nil {
		c.Error = err
	}

	if err := c.Container.Provide(repositories.NewResetTokenRepository); err != nil {
		c.Error = err
	}

	if err := c.Container.Provide(repositories.NewAuthSessionRepository); err != nil {
		c.Error = err
	}

	if err := c.Container.Provide(repositories.NewResumeRepository); err != nil {
		c.Error = err
	}

	if err := c.Container.Provide(repositories.NewInterviewSessionRepository); err != nil {
		c.Error = err
	}

	if err := c.Container.Provide(repositories.NewEvaluationScoresRepository); err != nil {
		c.Error = err
	}

	if err := c.Container.Provide(repositories.NewInterviewTurnsRepository); err != nil {
		c.Error = err
	}

	if err := c.Container.Provide(repositories.NewEvaluationRubricsRepository); err != nil {
		c.Error = err
	}

	if err := c.Container.Provide(repositories.NewIssueReportsRepository); err != nil {
		c.Error = err
	}

	if err := c.Container.Provide(repositories.NewIssueCategoriesRepository); err != nil {
		c.Error = err
	}

	if err := c.Container.Provide(repositories.NewReviewCommentRepository); err != nil {
		c.Error = err
	}
}
