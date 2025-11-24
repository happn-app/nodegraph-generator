package pkg

import (
	"log"
	"strings"

	"github.com/prometheus/common/model"
)

func ConnectionTypeToIcon(connectionType string, serviceName string) string {
	if serviceName == "user" {
		return "users-alt"
	}
	if serviceName == "google" {
		return "cloud"
	}
	if strings.HasSuffix(serviceName, "-api") {
		return "api-endpoint"
	}
	if strings.HasSuffix(serviceName, "-scheduler") {
		return "hourglass"
	}
	if strings.HasSuffix(serviceName, "-worker") {
		return "cog"
	}
	if strings.HasSuffix(serviceName, "-importer") {
		return "import"
	}

	switch connectionType {
	case "database":
		return "database"
	case "virtual_node":
		return "api-endpoint"
	default:
		log.Printf("Unknown connection type %q for service %q, using default icon", connectionType, serviceName)
		return "question-circle"
	}
}

func EdgeKey(sample *model.Sample) string {
	return MapServiceName(string(sample.Metric["client"])) + "-" + MapServiceName(string(sample.Metric["server"]))
}

func MapServiceName(serviceName string) string {
	return serviceName
}
