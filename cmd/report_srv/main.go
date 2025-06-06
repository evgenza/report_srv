package main

import (
	"report_srv/internal/di"
)

func main() {
	app := di.InitializeApp()
	app.Run()
}
