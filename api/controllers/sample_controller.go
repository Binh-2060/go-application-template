package controllers

import (
	"github.com/Binh-2060/go-application-template/api/presenters"
	"github.com/gofiber/fiber/v3"
)

func GetSampleController(c fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(presenters.ResponseSuccess("HELLO WORLD"))
}
