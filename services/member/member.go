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