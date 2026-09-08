// Command openapi writes Sharelock's generated OpenAPI document to a local file.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"

	"github.com/demeero/sharelock/internal/config"
	"github.com/demeero/sharelock/internal/share"
)

func main() {
	outputPath := flag.String("output", "frontend/openapi.json", "path for the OpenAPI JSON document")
	flag.Parse()

	if err := os.MkdirAll(filepath.Dir(*outputPath), 0o755); err != nil {
		fail(fmt.Errorf("create output directory: %w", err))
	}

	output, err := os.Create(*outputPath)
	if err != nil {
		fail(fmt.Errorf("create OpenAPI document: %w", err))
	}
	defer output.Close()

	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(openAPISpec()); err != nil {
		fail(fmt.Errorf("write OpenAPI document: %w", err))
	}
}

func openAPISpec() *huma.OpenAPI {
	mux := http.NewServeMux()
	api := humago.New(mux, huma.DefaultConfig("Sharelock API", "1.0.0"))
	apiGroup := huma.NewGroup(api, "/api/v1/shares")
	share.RegisterRoutes(apiGroup, nil, config.ShareConfig{})

	return api.OpenAPI()
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
