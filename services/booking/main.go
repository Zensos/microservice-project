package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v3"
	"github.com/zensos/microservice-project/internal/common"
)

func main() {
	app := fiber.New()

	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("Booking Service")
	})

	app.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "booking"})
	})
	
	// Note: Dont touch this naja
	consulClient, serviceID, err := common.RegisterService(common.ServiceConfig{
		Name: "booking",
		Port: 3001,
	})
	if err != nil {
		log.Printf("couldn't register with consul: %v", err)
	}

	// TODO: replace it with real business logic
	app.Post("/bookings", func(c fiber.Ctx) error {
		if consulClient == nil {
			return c.Status(503).JSON(fiber.Map{"error": "service discovery is not available right now"})
		}

		eventAddr, err := common.DiscoverService(consulClient, "event")
		if err != nil {
			return c.Status(502).JSON(fiber.Map{"error": fmt.Sprintf("couldn't find the event service: %v", err)})
		}

		eventResp, err := http.Get(fmt.Sprintf("http://%s/events/1", eventAddr))
		if err != nil {
			return c.Status(502).JSON(fiber.Map{"error": fmt.Sprintf("couldn't reach the event service: %v", err)})
		}
		defer eventResp.Body.Close()

		var eventData map[string]any
		eventBody, _ := io.ReadAll(eventResp.Body)
		if err := json.Unmarshal(eventBody, &eventData); err != nil {
			return c.Status(502).JSON(fiber.Map{"error": "got a bad response from the event service"})
		}

		paymentAddr, err := common.DiscoverService(consulClient, "payment")
		if err != nil {
			return c.Status(502).JSON(fiber.Map{"error": fmt.Sprintf("couldn't find the payment service: %v", err)})
		}

		paymentReq, _ := json.Marshal(map[string]any{
			"booking_id": "book_123",
			"amount":     eventData["price"],
		})

		paymentResp, err := http.Post(
			fmt.Sprintf("http://%s/payments", paymentAddr),
			"application/json",
			bytes.NewReader(paymentReq),
		)
		if err != nil {
			return c.Status(502).JSON(fiber.Map{"error": fmt.Sprintf("couldn't reach the payment service: %v", err)})
		}
		defer paymentResp.Body.Close()

		var paymentData map[string]any
		paymentBody, _ := io.ReadAll(paymentResp.Body)
		if err := json.Unmarshal(paymentBody, &paymentData); err != nil {
			return c.Status(502).JSON(fiber.Map{"error": "got a bad response from the payment service"})
		}

		return c.JSON(fiber.Map{
			"booking_id": "book_123",
			"event":      eventData,
			"payment":    paymentData,
			"status":     "confirmed",
		})
	})

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

	log.Fatal(app.Listen(":3001"))
}
