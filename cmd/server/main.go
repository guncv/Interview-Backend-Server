package main

import (
	"gitlab.com/interview-simulation/interview-backend-server/internal/containers"
	_ "gitlab.com/interview-simulation/interview-backend-server/internal/handlers"
)

// @title           Interview Simulation API
// @version         1.0
// @description     A backend server for interview simulation platform
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      ${API_HOST:-localhost:8080}
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
func main() {
	c := containers.NewContainer()
	if err := c.Run().Error; err != nil {
		panic(err)
	}
}
