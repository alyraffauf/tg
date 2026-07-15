package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/spf13/cobra"
)

var (
	apiMethod string
	apiFields []string
)

var apiCmd = &cobra.Command{
	Use:   "api <nsid>",
	Short: "Call an authenticated XRPC endpoint",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if auth == nil || !auth.IsAuthenticated() {
			return fmt.Errorf("not logged in; run \"tg auth login\" first")
		}

		endpoint, err := syntax.ParseNSID(args[0])
		if err != nil {
			return fmt.Errorf("parse NSID: %w", err)
		}
		fields, err := parseAPIFields(apiFields)
		if err != nil {
			return err
		}
		method := strings.ToUpper(apiMethod)
		if method == "GET" && len(apiFields) > 0 && !cmd.Flags().Changed("method") {
			method = "POST"
		}
		if method != http.MethodGet && method != http.MethodPost {
			return fmt.Errorf("method must be GET or POST, got %q", apiMethod)
		}

		client, err := auth.APIClient(cmd.Context())
		if err != nil {
			return fmt.Errorf("get auth client: %w", err)
		}
		response, err := doAPIRequest(cmd, client, endpoint, method, fields)
		if err != nil {
			return err
		}
		defer response.Body.Close()
		return writeAPIResponse(cmd, response)
	},
}

func init() {
	apiCmd.Flags().StringVarP(&apiMethod, "method", "X", "GET", "HTTP method (GET or POST)")
	apiCmd.Flags().StringArrayVarP(&apiFields, "field", "f", nil, "Add a key=value field")
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

func doAPIRequest(cmd *cobra.Command, client *atclient.APIClient, endpoint syntax.NSID, method string, fields map[string]any) (*http.Response, error) {
	request := atclient.NewAPIRequest(method, endpoint, nil)
	if method == http.MethodGet {
		query := make(map[string]string, len(fields))
		for key, value := range fields {
			query[key] = fmt.Sprint(value)
		}
		request.QueryParams = makeURLValues(query)
	} else {
		encoded, err := json.Marshal(fields)
		if err != nil {
			return nil, fmt.Errorf("encode request body: %w", err)
		}
		request.Body = bytes.NewReader(encoded)
		request.Headers.Set("Content-Type", "application/json")
	}
	request.Headers.Set("Accept", "application/json")

	response, err := client.Do(cmd.Context(), request)
	if err != nil {
		return nil, fmt.Errorf("call %s: %w", endpoint, err)
	}
	return response, nil
}

func makeURLValues(fields map[string]string) map[string][]string {
	values := make(map[string][]string, len(fields))
	for key, value := range fields {
		values[key] = []string{value}
	}
	return values
}

func writeAPIResponse(cmd *cobra.Command, response *http.Response) error {
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("read API response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("API returned HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}

	if json.Valid(body) {
		var formatted bytes.Buffer
		if err := json.Indent(&formatted, body, "", "  "); err == nil {
			formatted.WriteByte('\n')
			_, err = cmd.OutOrStdout().Write(formatted.Bytes())
			return err
		}
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}
