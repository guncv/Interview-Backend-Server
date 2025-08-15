package main

import "gitlab.com/interview-simulation/interview-backend-server/internal/containers"

func main() {
	c := containers.NewContainer()
	if err := c.Run().Error; err != nil {
		panic(err)
	}
}
