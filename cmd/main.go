// cmd/main.go
package main

import (
	"go-bank-api/app"
)

// @title           Go-Bank API
// @version         1.0
// @description     This is a sample banking API built with Go.

// @contact.name   API Support
// @contact.email  support@example.com

// @license.name   MIT
// @license.url    https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	app.Run()
}
