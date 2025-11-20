package rundeck

import (
	"context"

	"github.com/rundeck/go-rundeck/rundeck"
	openapi "github.com/rundeck/go-rundeck/rundeck-v2"
)

// RundeckClients holds both v1 and v2 Rundeck API clients
// This struct is used throughout the provider to interact with different versions
// of the Rundeck API
type RundeckClients struct {
	V1         *rundeck.BaseClient
	V2         *openapi.APIClient
	Token      string
	BaseURL    string
	APIVersion string
	ctx        context.Context
}
