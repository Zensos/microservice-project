package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v3"
	"github.com/zensos/microservice-project/internal/common"
)

// เพศ
type Gender string

const (
	GenderMale   Gender = "male"
	GenderFemale Gender = "female"
	GenderOther  Gender = "other"
	GenderNA     Gender = "not_specified"
)

// ประเภทเอกสารระบุตัวตน
type IdentityType string

const (
	IdentityNationalID IdentityType = "national_id"
	IdentityPassport   IdentityType = "passport"
)

// วัน-เดือน-ปี เกิด
type DateOfBirth struct {
	Day   int `json:"day"`
	Month int `json:"month"`
	Year  int `json:"year"`
}

// เบอร์โทรศัพท์
type PhoneNumber struct {
	CountryCode string `json:"country_code"`
	Number      string `json:"number"`
}

// ที่อยู่
type Address struct {
	Line1      string `json:"line1"`
	Country    string `json:"country"`
	Province   string `json:"province"`
	District   string `json:"district"`
	PostalCode string `json:"postal_code"`
}

type Member struct {
	ID           int          `json:"id"`
	FirstName    string       `json:"first_name"`
	LastName     string       `json:"last_name"`
	Email        string       `json:"email"`
	PasswordHash string       `json:"-"`
	Gender       Gender       `json:"gender,omitempty"`
	DateOfBirth  DateOfBirth  `json:"date_of_birth,omitempty"`
	Phone        PhoneNumber  `json:"phone,omitempty"`
	Address      Address      `json:"address,omitempty"`
	IdentityType IdentityType `json:"identity_type,omitempty"`
}

type UpdateMemberRequest struct {
	FirstName   *string      `json:"first_name,omitempty"`
	LastName    *string      `json:"last_name,omitempty"`
	Gender      *Gender      `json:"gender,omitempty"`
	DateOfBirth *DateOfBirth `json:"date_of_birth,omitempty"`
	Address     *Address     `json:"address,omitempty"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
}

type SignUpRequest struct {
	Email           string `json:"email"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
}

type SignInRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// mock member data
var members = []Member{
	{
		ID:           1,
		FirstName:    "John",
		LastName:     "Doe",
		Email:        "john@gmail.com",
		PasswordHash: "john",
		Gender:       GenderMale,
		DateOfBirth: DateOfBirth{
			Day:   1,
			Month: 1,
			Year:  1990,
		},
		Phone: PhoneNumber{
			CountryCode: "+66",
			Number:      "123456789",
		},
		Address: Address{
			Line1:      "123 Main St",
			Country:    "Thailand",
			Province:   "Bangkok",
			District:   "Sathon",
			PostalCode: "10120",
		},
		IdentityType: IdentityNationalID,
	},
}

func main() {
	app := fiber.New()

	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("Member Service")
	})

	app.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "member"})
	})

	// Rgister with Consul for service discovery
	consulClient, serviceID, err := common.RegisterService(common.ServiceConfig{
		Name: "member",
		Port: 3003,
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

	// GET : member by id
	app.Get("/members/:id", getMemberProfile)
	// Patch : update member profile
	app.Patch("/members/:id", updateMemberProfile)
	// POST : change password
	app.Post("/members/:id/change-password", changePassword)
	// POST : signup
	app.Post("/auth/signup", signup)
	// POST : signin
	app.Post("/auth/signin", signin)

	log.Fatal(app.Listen(":3003"))
}
