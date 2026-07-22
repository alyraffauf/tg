package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/spf13/cobra"
)

func newAPICommand(service *app.Service) *cobra.Command {
	var methodFlag string
	var fieldsFlag []string

	command := &cobra.Command{
		Use:   "api <nsid>",
		Short: "Call an authenticated XRPC endpoint",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			endpoint, err := syntax.ParseNSID(args[0])
			if err != nil {
				return fmt.Errorf("parse NSID: %w", err)
			}
			fields, err := parseAPIFields(fieldsFlag)
			if err != nil {
				return err
			}
			method := strings.ToUpper(methodFlag)
			if method == "GET" && len(fieldsFlag) > 0 && !cmd.Flags().Changed("method") {
				method = "POST"
			}
			if method != http.MethodGet && method != http.MethodPost {
				return fmt.Errorf("method must be GET or POST, got %q", methodFlag)
			}

			response, err := service.CallAPI(cmd.Context(), app.APIRequestInput{
				Endpoint: endpoint,
				Method:   method,
				Fields:   fields,
			})
			if err != nil {
				return err
			}
			return writeAPIResponse(cmd, response)
		},
	}
	command.Flags().StringVarP(&methodFlag, "method", "X", "GET", "HTTP method (GET or POST)")
	command.Flags().StringArrayVarP(&fieldsFlag, "field", "f", nil, "Add a key=value field")
	return command
}

func parseAPIFields(rawFields []string) (map[string]any, error) {
	fields := make(map[string]any, len(rawFields))
	for _, rawField := range rawFields {
		key, value, found := strings.Cut(rawField, "=")
		if !found || key == "" {
			return nil, fmt.Errorf("field must be key=value, got %q", rawField)
		}

		var decoded any
		if err := json.Unmarshal([]byte(value), &decoded); err == nil {
			fields[key] = decoded
		} else {
			fields[key] = value
		}
	}
	return fields, nil
}

func writeAPIResponse(cmd *cobra.Command, response *app.APIResponse) error {
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("API returned HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(response.Body)))
	}

	if json.Valid(response.Body) {
		var formatted bytes.Buffer
		if err := json.Indent(&formatted, response.Body, "", "  "); err == nil {
			formatted.WriteByte('\n')
			_, err = cmd.OutOrStdout().Write(formatted.Bytes())
			return err
		}
	}
	_, err := cmd.OutOrStdout().Write(response.Body)
	return err
}
