package tangledlex

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/rivo/uniseg"
)

//go:embed schemas/org
var schemaFiles embed.FS

// ValidateRecord checks outbound records against the pinned upstream lexicons.
func ValidateRecord(collection string, record any) error {
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	object, ok := value.(map[string]any)
	if !ok || object["$type"] != collection {
		return fmt.Errorf("$type must be %q", collection)
	}
	return validateRef(collection, "", value, 0)
}

func loadDefinition(nsid, definition string) (map[string]any, error) {
	data, err := schemaFiles.ReadFile("schemas/" + strings.ReplaceAll(nsid, ".", "/") + ".json")
	if err != nil {
		return nil, fmt.Errorf("no pinned schema for %s: %w", nsid, err)
	}
	var schema struct {
		Defs map[string]map[string]any `json:"defs"`
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, err
	}
	if definition == "" {
		definition = "main"
	}
	found := schema.Defs[definition]
	if found == nil {
		return nil, fmt.Errorf("missing definition %s#%s", nsid, definition)
	}
	return found, nil
}
func validateRef(reference, base string, value any, depth int) error {
	nsid, definition, _ := strings.Cut(reference, "#")
	if nsid == "" {
		nsid = base
	}
	if nsid == "com.atproto.repo.strongRef" {
		object, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("strong reference must be an object")
		}
		uri, _ := object["uri"].(string)
		cid, _ := object["cid"].(string)
		if _, err := syntax.ParseATURI(uri); err != nil {
			return err
		}
		_, err := syntax.ParseCID(cid)
		return err
	}
	if nsid == "com.atproto.label.defs" {
		return nil
	}
	schema, err := loadDefinition(nsid, definition)
	if err != nil {
		return err
	}
	return validateValue(schema, nsid, value, depth+1)
}
func validateValue(schema map[string]any, nsid string, value any, depth int) error {
	if depth > 64 {
		return fmt.Errorf("record nesting exceeds 64 levels")
	}
	kind, _ := schema["type"].(string)
	switch kind {
	case "record":
		return validateValue(schema["record"].(map[string]any), nsid, value, depth+1)
	case "object":
		object, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("expected object")
		}
		required, _ := schema["required"].([]any)
		for _, field := range required {
			if object[field.(string)] == nil {
				return fmt.Errorf("%s is required", field)
			}
		}
		properties, _ := schema["properties"].(map[string]any)
		for key, child := range properties {
			if member, ok := object[key]; ok && member != nil {
				if err := validateValue(child.(map[string]any), nsid, member, depth+1); err != nil {
					return fmt.Errorf("%s: %w", key, err)
				}
			}
		}
	case "ref":
		return validateRef(schema["ref"].(string), nsid, value, depth+1)
	case "union":
		object, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("expected tagged object")
		}
		tag, _ := object["$type"].(string)
		refs, _ := schema["refs"].([]any)
		for _, reference := range refs {
			ref := reference.(string)
			if strings.HasPrefix(ref, "#") {
				ref = nsid + ref
			}
			if tag == ref || strings.TrimSuffix(tag, "#main") == ref {
				return validateRef(ref, nsid, value, depth+1)
			}
		}
		return fmt.Errorf("unexpected union type %q", tag)
	case "string":
		text, ok := value.(string)
		if !ok {
			return fmt.Errorf("expected string")
		}
		if !utf8.ValidString(text) {
			return fmt.Errorf("invalid UTF-8")
		}
		length := float64(len(text))
		graphemes := float64(uniseg.GraphemeClusterCount(text))
		for key, actual := range map[string]float64{"minLength": length, "maxLength": length, "minGraphemes": graphemes, "maxGraphemes": graphemes} {
			if bound, ok := schema[key].(float64); ok && ((strings.HasPrefix(key, "min") && actual < bound) || (strings.HasPrefix(key, "max") && actual > bound)) {
				return fmt.Errorf("does not satisfy %s %g", key, bound)
			}
		}
		format, _ := schema["format"].(string)
		var err error
		switch format {
		case "did":
			_, err = syntax.ParseDID(text)
		case "at-uri":
			_, err = syntax.ParseATURI(text)
		case "cid":
			_, err = syntax.ParseCID(text)
		case "record-key":
			_, err = syntax.ParseRecordKey(text)
		case "datetime":
			_, err = syntax.ParseDatetime(text)
		case "uri":
			var u *url.URL
			u, err = url.Parse(text)
			if err == nil && u.Scheme == "" {
				err = fmt.Errorf("expected absolute URI")
			}
		}
		if err != nil {
			return err
		}
	case "array":
		values, ok := value.([]any)
		if !ok {
			return fmt.Errorf("expected array")
		}
		if max, ok := schema["maxLength"].(float64); ok && float64(len(values)) > max {
			return fmt.Errorf("too many items")
		}
		if min, ok := schema["minLength"].(float64); ok && float64(len(values)) < min {
			return fmt.Errorf("too few items")
		}
		item, _ := schema["items"].(map[string]any)
		for _, member := range values {
			if err := validateValue(item, nsid, member, depth+1); err != nil {
				return err
			}
		}
	case "blob":
		object, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("expected blob")
		}
		ref, _ := object["ref"].(map[string]any)
		cid, _ := ref["$link"].(string)
		if _, err := syntax.ParseCID(cid); err != nil {
			return err
		}
		size, ok := object["size"].(float64)
		if !ok || size < 0 {
			return fmt.Errorf("invalid blob size")
		}
		if max, ok := schema["maxSize"].(float64); ok && size > max {
			return fmt.Errorf("blob exceeds %g bytes", max)
		}
		mime, _ := object["mimeType"].(string)
		if accepted, ok := schema["accept"].([]any); ok {
			allowed := false
			for _, pattern := range accepted {
				candidate := pattern.(string)
				if candidate == mime || candidate == "*/*" || (strings.HasSuffix(candidate, "/*") && strings.HasPrefix(mime, strings.TrimSuffix(candidate, "*"))) {
					allowed = true
				}
			}
			if !allowed {
				return fmt.Errorf("unsupported blob media type %q", mime)
			}
		}
	}
	return nil
}
