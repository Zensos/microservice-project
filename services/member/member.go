package main

import (
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v3"
)

func getMemberProfile(c fiber.Ctx) error {
	// TODO: replace with real business logic
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}
	

	for _, member := range members {
		if member.ID == id {
			return c.JSON(member)
		}
	}

	return c.Status(fiber.StatusBadRequest).SendString("Member with ID " + c.Params("id") + " not found")
}

func updateMemberProfile(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}

	memberUpdate := new(UpdateMemberRequest)
	if err := c.Bind().JSON(memberUpdate); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
	}

	for i, member := range members {
		fmt.Println(member, i)
		if member.ID == id {
			if memberUpdate.FirstName != nil {
				member.FirstName = *memberUpdate.FirstName
			}
			if memberUpdate.LastName != nil {
				member.LastName = *memberUpdate.LastName
			}
			if memberUpdate.Gender != nil {
				member.Gender = *memberUpdate.Gender
			}
			if memberUpdate.DateOfBirth != nil {
				member.DateOfBirth = *memberUpdate.DateOfBirth
			}
			if memberUpdate.Address != nil {
				member.Address = *memberUpdate.Address
			}
			members[i] = member
			return c.JSON(member)
		}
	}

	return c.Status(fiber.StatusNotFound).SendString("Member with ID " + c.Params("id") + " not found")
}

func changePassword(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}
	
	changePassword := new(ChangePasswordRequest)
	if err := c.Bind().JSON(changePassword); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
	}
	for i, member := range members {
		if member.ID == id {
			if member.PasswordHash != changePassword.CurrentPassword {
				return c.Status(fiber.StatusBadRequest).SendString("Current password is incorrect")
			}
			if changePassword.NewPassword != changePassword.ConfirmPassword {
				return c.Status(fiber.StatusBadRequest).SendString("New password and confirm password do not match")
			}
			member.PasswordHash = changePassword.NewPassword
			members[i] = member
			return c.JSON(fiber.Map{"message": "Password changed successfully"})
		}
	}

	return c.Status(fiber.StatusNotFound).SendString("Member with ID " + c.Params("id") + " not found")
}

func signup(c fiber.Ctx) error {
	newMember := new(SignUpRequest)
	if err := c.Bind().JSON(newMember); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}

	// password and confirm password must match
	if newMember.Password != newMember.ConfirmPassword {
		return c.Status(fiber.StatusBadRequest).SendString("Password and confirm password do not match")
	}

	// check if email already exists
	for _, m := range members {
		if m.Email == newMember.Email {
			return c.Status(fiber.StatusBadRequest).SendString("Email already exists")
		}
	}

	members = append(members, Member{
		ID:        len(members) + 1,
		FirstName: newMember.FirstName,
		LastName:  newMember.LastName,
		Email:     newMember.Email,
		PasswordHash: newMember.Password,
	})
	return c.JSON(newMember)
}

func signin(c fiber.Ctx) error {
	member := new(SignInRequest)
	if err := c.Bind().JSON(member); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}
	for _, m := range members {
		if m.Email == member.Email && m.PasswordHash == member.Password {
			return c.JSON(fiber.Map{
				"message": "Sign in successful",
				"member":  m,
			})
		}
	}
	return c.Status(fiber.StatusUnauthorized).SendString("Invalid email or password")
}