package main

import (
	"fhir-bomber/pkg/client"
	"fhir-bomber/pkg/config"
	"fhir-bomber/pkg/monitoring"
)

func main() {
	appConfig := config.LoadConfig()
	m := monitoring.Setup()
	go monitoring.Run(m.Registry)

	bomber := client.NewBomber(appConfig, m)
	bomber.Run()
}
