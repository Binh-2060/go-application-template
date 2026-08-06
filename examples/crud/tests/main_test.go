//go:build integration

/*
Package tests holds every test for the users CRUD example.

The tests live outside the packages they exercise, so they see the same exported
API a caller would. Two consequences worth knowing:

  - Anything a test needs must be exported. repositories.Builder,
    ApplyUserFilter and BuildListUsersQuery exist in the public API only because
    this package asserts against them; in-package test files would not have
    needed that.
  - A test cannot reach an unexported constant, so values like the pagination
    defaults are written as literals here and the assertion says where the real
    definition lives.

This file is built only under -tags=integration, so an untagged `go test` runs
the SQL-generation tests with no TestMain and no database.
*/
package tests

import (
	"testing"

	"github.com/Binh-2060/go-application-template/examples/crud/testsupport"
)

func TestMain(m *testing.M) { testsupport.Main(m) }

// A unique name prefix for this test, with row cleanup registered.
func marker(t *testing.T) string { return testsupport.Marker(t) }
