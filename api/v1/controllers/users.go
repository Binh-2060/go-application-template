package controllers

import (
	"github.com/Binh-2060/go-application-template/api/presenters"
	"github.com/Binh-2060/go-application-template/api/v1/services"
	"github.com/gofiber/fiber/v2"
)

func GetUserController(c *fiber.Ctx) error {
	result, err := services.GetUserService(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusOK).JSON(presenters.ResponseSuccess(result))
}
