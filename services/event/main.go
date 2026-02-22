package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v3"
	"github.com/zensos/microservice-project/internal/common"
)

func main() {
	app := fiber.New()

	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("Event Service")
	})

	app.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "event"})
	})

	// TODO: replace it with real business logic
	app.Get("/events/:id", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"id":    1,
			"name":  "Go Conference 2026",
			"price": 150.00,
		})
	})

	// Note: Dont touch this naja
	consulClient, serviceID, err := common.RegisterService(common.ServiceConfig{
		Name: "event",
		Port: 3002,
	})
	if err != nil {
		log.Printf("Warning: failed to register with Consul: %v", err)
	}

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		if consulClient != nil {
			if err := common.DeregisterService(consulClient, serviceID); err != nil {
				log.Printf("Warning: failed to deregister from Consul: %v", err)
			}
		}
		if err := app.Shutdown(); err != nil {
			log.Printf("Warning: server shutdown error: %v", err)
		}
	}()

	log.Fatal(app.Listen(":3002"))
}
