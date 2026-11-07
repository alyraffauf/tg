package app

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/alyraffauf/tg/tangled"
)

// Label is a repository label definition, or a definition applied to an issue
// or pull request. An applied label whose definition no longer exists keeps
// its definition URI and has an empty Name.
type Label struct {
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
	URI   string `json:"uri"`
}

// LabelEditInput names labels to apply to or remove from a ticket. Names must
// match exactly one repository label definition.
type LabelEditInput struct {
	AddLabels    []string
	RemoveLabels []string
}

func (input LabelEditInput) isEmpty() bool {
	return len(input.AddLabels) == 0 && len(input.RemoveLabels) == 0
}

var (
	// ErrIncompleteLabelCatalog prevents name resolution against partial definitions.
	ErrIncompleteLabelCatalog = errors.New("label definitions are incomplete")
	// ErrUnknownLabel means no repository label definition has the given name.
	ErrUnknownLabel = errors.New("unknown label")
	// ErrAmbiguousLabel means several repository label definitions share the
	// given name, so an edit cannot tell which one to apply or remove.
	ErrAmbiguousLabel = errors.New("ambiguous label")
	// ErrConflictingLabelEdit means one edit both adds and removes a label.
	ErrConflictingLabelEdit = errors.New("label is both added and removed")
)

// labelCatalog indexes one repository's label definitions by record key and
// by name. Definition names are not unique upstream.
type labelCatalog struct {
	repoDID     string
	definitions []tangled.LabelDefinition
	byKey       map[string]tangled.LabelDefinition
	keysByName  map[string][]string
	// available is false when the definitions could not be read; applied
	// labels then render as URIs without per-label warnings.
	available bool
}

func newLabelCatalog(repoDID string, definitions []tangled.LabelDefinition) labelCatalog {
	catalog := labelCatalog{
		repoDID:     repoDID,
		definitions: definitions,
		byKey:       make(map[string]tangled.LabelDefinition, len(definitions)),
		keysByName:  make(map[string][]string, len(definitions)),
		available:   true,
	}
	for _, definition := range definitions {
		catalog.byKey[definition.Key] = definition
		catalog.keysByName[definition.Name] = append(catalog.keysByName[definition.Name], definition.Key)
	}
	return catalog
}

func unavailableLabelCatalog(repoDID string) labelCatalog {
	catalog := newLabelCatalog(repoDID, nil)
	catalog.available = false
	return catalog
}

// loadLabelCatalog reads every label definition of a repository.
func (s *Service) loadLabelCatalog(ctx context.Context, repoDID string) (labelCatalog, []RecordWarning, error) {
	list, err := s.appview.ListLabelDefinitions(ctx, repoDID)
	if err != nil {
		return labelCatalog{}, nil, err
	}
	return newLabelCatalog(repoDID, list.Items), recordWarnings(list.Warnings), nil
}

func (s *Service) loadCompleteLabelCatalog(ctx context.Context, repoDID string) (labelCatalog, error) {
	catalog, warnings, err := s.loadLabelCatalog(ctx, repoDID)
	if err != nil {
		return labelCatalog{}, err
	}
	if len(warnings) > 0 {
		return labelCatalog{}, fmt.Errorf("%w: %s", ErrIncompleteLabelCatalog, warnings[0].Error)
	}
	return catalog, nil
}

// loadDisplayLabelCatalog reads label definitions for rendering. Reads that
// only display labels still succeed when definitions are unavailable.
func (s *Service) loadDisplayLabelCatalog(ctx context.Context, repoDID string) (labelCatalog, []RecordWarning) {
	catalog, warnings, err := s.loadLabelCatalog(ctx, repoDID)
	if err != nil {
		return unavailableLabelCatalog(repoDID), []RecordWarning{{Error: fmt.Sprintf("label names unavailable: %v", err)}}
	}
	return catalog, warnings
}

func (c labelCatalog) definitionLabels() []Label {
	labels := make([]Label, 0, len(c.definitions))
	for _, definition := range c.definitions {
		labels = append(labels, labelFromDefinition(definition))
	}
	return labels
}

func labelFromDefinition(definition tangled.LabelDefinition) Label {
	return Label{Name: definition.Name, Color: definition.Color, URI: definition.URI}
}

func (c labelCatalog) definitionURI(key string) string {
	return "at://" + c.repoDID + "/" + tangled.LabelDefinitionCollection + "/" + key
}

// appliedLabels renders a ticket's label tokens in their stored order, once
// per definition. It returns the keys that have no definition.
func (c labelCatalog) appliedLabels(tokens []tangled.LabelToken) ([]Label, []string) {
	labels := make([]Label, 0, len(tokens))
	var unresolvedKeys []string
	for _, key := range appliedLabelKeys(tokens) {
		definition, found := c.byKey[key]
		if !found {
			labels = append(labels, Label{URI: c.definitionURI(key)})
			if c.available {
				unresolvedKeys = append(unresolvedKeys, key)
			}
			continue
		}
		labels = append(labels, labelFromDefinition(definition))
	}
	return labels, unresolvedKeys
}

func appliedLabelKeys(tokens []tangled.LabelToken) []string {
	keys := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if (token.Type != "" && token.Type != tangled.LabelTokenType) || token.Key == "" || slices.Contains(keys, token.Key) {
			continue
		}
		keys = append(keys, token.Key)
	}
	return keys
}

func unresolvedLabelMessage(keys []string) string {
	return fmt.Sprintf("applied label definitions not found: %s", strings.Join(keys, ", "))
}

// filterKeySets resolves filter names to the definition keys that satisfy
// each name. A shared name matches a ticket carrying any of its definitions.
func (c labelCatalog) filterKeySets(names []string) ([][]string, error) {
	keySets := make([][]string, 0, len(names))
	for _, name := range names {
		keys := c.keysByName[name]
		if len(keys) == 0 {
			return nil, fmt.Errorf("%w %q", ErrUnknownLabel, name)
		}
		keySets = append(keySets, keys)
	}
	return keySets, nil
}

// editKey resolves a name to the one definition an edit applies or removes.
func (c labelCatalog) editKey(name string) (string, error) {
	keys := c.keysByName[name]
	switch len(keys) {
	case 0:
		return "", fmt.Errorf("%w %q", ErrUnknownLabel, name)
	case 1:
		return keys[0], nil
	default:
		uris := make([]string, 0, len(keys))
		for _, key := range keys {
			uris = append(uris, c.definitionURI(key))
		}
		return "", fmt.Errorf("%w %q matches %s", ErrAmbiguousLabel, name, strings.Join(uris, ", "))
	}
}

// matchesAllLabels reports whether tokens satisfy every key set.
func matchesAllLabels(tokens []tangled.LabelToken, keySets [][]string) bool {
	applied := appliedLabelKeys(tokens)
	for _, keys := range keySets {
		if !slices.ContainsFunc(keys, func(key string) bool { return slices.Contains(applied, key) }) {
			return false
		}
	}
	return true
}

// checkLabelEditConflicts rejects an edit that names one label on both sides.
func checkLabelEditConflicts(input LabelEditInput) error {
	for _, name := range input.AddLabels {
		if slices.Contains(input.RemoveLabels, name) {
			return fmt.Errorf("%w: %q", ErrConflictingLabelEdit, name)
		}
	}
	return nil
}

type labelKeyEdit struct {
	add    []string
	remove []string
}

func (c labelCatalog) resolveLabelEdit(input LabelEditInput) (labelKeyEdit, error) {
	var edit labelKeyEdit
	for _, name := range input.AddLabels {
		key, err := c.editKey(name)
		if err != nil {
			return labelKeyEdit{}, err
		}
		edit.add = append(edit.add, key)
	}
	for _, name := range input.RemoveLabels {
		key, err := c.editKey(name)
		if err != nil {
			return labelKeyEdit{}, err
		}
		edit.remove = append(edit.remove, key)
	}
	return edit, nil
}

// applyLabelKeyEdit rewrites a ticket record's tangledLabels in place. It
// drops duplicate label tokens and omits the field when no labels remain.
// Non-token entries pass through unchanged so outbound validation can reject
// unknown union types. It reports whether the stored value changed.
func applyLabelKeyEdit(record map[string]any, edit labelKeyEdit) bool {
	existing, _ := record["tangledLabels"].([]any)
	updated := make([]any, 0, len(existing)+len(edit.add))
	var keptKeys []string
	changed := false
	for _, entry := range existing {
		key, isToken := labelTokenKey(entry)
		if !isToken {
			updated = append(updated, entry)
			continue
		}
		if slices.Contains(edit.remove, key) || slices.Contains(keptKeys, key) {
			changed = true
			continue
		}
		keptKeys = append(keptKeys, key)
		updated = append(updated, entry)
	}
	for _, key := range edit.add {
		if slices.Contains(keptKeys, key) {
			continue
		}
		keptKeys = append(keptKeys, key)
		changed = true
		updated = append(updated, map[string]any{"$type": tangled.LabelTokenType, "key": key})
	}

	if len(updated) == 0 {
		delete(record, "tangledLabels")
	} else {
		record["tangledLabels"] = updated
	}
	return changed
}

func labelTokenKey(entry any) (string, bool) {
	token, _ := entry.(map[string]any)
	if tokenType, present := token["$type"]; present && tokenType != tangled.LabelTokenType {
		return "", false
	}
	key, _ := token["key"].(string)
	return key, key != ""
}

// Knot serializes label tokens without a union discriminator. Restore the
// discriminator for the local outbound validator without accepting unknown types.
func normalizeTicketLabelTokens(record map[string]any) {
	entries, _ := record["tangledLabels"].([]any)
	for _, entry := range entries {
		if _, known := labelTokenKey(entry); known {
			entry.(map[string]any)["$type"] = tangled.LabelTokenType
		}
	}
}

// recordLabelKeys returns the label keys stored in a raw ticket record.
func recordLabelKeys(record map[string]any) []string {
	entries, _ := record["tangledLabels"].([]any)
	var keys []string
	for _, entry := range entries {
		if key, isToken := labelTokenKey(entry); isToken && !slices.Contains(keys, key) {
			keys = append(keys, key)
		}
	}
	return keys
}
