package common

import (
	"fmt"
	"log"
	"os"
	"time"

	consul "github.com/hashicorp/consul/api"
)

type ServiceConfig struct {
	Name string
	Port int
}

func RegisterService(cfg ServiceConfig) (*consul.Client, string, error) {
	consulAddr := os.Getenv("CONSUL_ADDR")
	if consulAddr == "" {
		consulAddr = "localhost:8500"
	}

	serviceHost := os.Getenv("SERVICE_HOST")
	if serviceHost == "" {
		serviceHost = "localhost"
	}

	config := consul.DefaultConfig()
	config.Address = consulAddr

	client, err := consul.NewClient(config)
	if err != nil {
		return nil, "", fmt.Errorf("couldn't create consul client: %w", err)
	}

	serviceID := fmt.Sprintf("%s-%s-%d", cfg.Name, serviceHost, cfg.Port)

	registration := &consul.AgentServiceRegistration{
		ID:      serviceID,
		Name:    cfg.Name,
		Address: serviceHost,
		Port:    cfg.Port,
		Check: &consul.AgentServiceCheck{
			HTTP:                           fmt.Sprintf("http://%s:%d/health", serviceHost, cfg.Port),
			Interval:                       "10s",
			Timeout:                        "5s",
			DeregisterCriticalServiceAfter: "30s",
		},
	}

	var regErr error
	for i := range 10 {
		if regErr = client.Agent().ServiceRegister(registration); regErr == nil {
			return client, serviceID, nil
		}
		log.Printf("consul registration attempt %d/10 failed: %v", i+1, regErr)
		time.Sleep(2 * time.Second)
	}

	return nil, "", fmt.Errorf("gave up registering after 10 attempts: %w", regErr)
}

func DeregisterService(client *consul.Client, serviceID string) error {
	return client.Agent().ServiceDeregister(serviceID)
}

func DiscoverService(client *consul.Client, name string) (string, error) {
	services, _, err := client.Health().Service(name, "", true, nil)
	if err != nil {
		return "", fmt.Errorf("couldn't discover %s service: %w", name, err)
	}
	if len(services) == 0 {
		return "", fmt.Errorf("no healthy %s instances found", name)
	}

	svc := services[0]
	return fmt.Sprintf("%s:%d", svc.Service.Address, svc.Service.Port), nil
}
