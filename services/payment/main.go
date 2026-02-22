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
		return c.SendString("Payment Service")
	})

	app.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "payment"})
	})

	// TODO: replace it with real business logic
	app.Post("/payments", func(c fiber.Ctx) error {
		var req struct {
			BookingID string  `json:"booking_id"`
			Amount    float64 `json:"amount"`
		}
		if err := c.Bind().JSON(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "couldn't understand the request, please check the format"})
		}
		return c.JSON(fiber.Map{
			"id":         "pay_001",
			"booking_id": req.BookingID,
			"amount":     req.Amount,
			"status":     "confirmed",
		})
	})

	// Note: Dont touch this naja
	consulClient, serviceID, err := common.RegisterService(common.ServiceConfig{
		Name: "payment",
		Port: 3004,
	})
	if err != nil {
		log.Printf("couldn't register with consul: %v", err)
	}

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		if consulClient != nil {
			if err := common.DeregisterService(consulClient, serviceID); err != nil {
				log.Printf("couldn't deregister from consul: %v", err)
			}
		}
		if err := app.Shutdown(); err != nil {
			log.Printf("something went wrong while shutting down: %v", err)
		}
	}()

	log.Fatal(app.Listen(":3004"))
}
