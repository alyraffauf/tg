package tangled

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/bluesky-social/indigo/lex/util"
)

func recordJSON(value *util.LexiconTypeDecoder, expected any) (json.RawMessage, error) {
	if value == nil || value.Val == nil {
		return nil, fmt.Errorf("Bobbin response is missing a record value")
	}
	if reflect.TypeOf(value.Val) != reflect.TypeOf(expected) {
		return nil, fmt.Errorf("Bobbin response has record type %T; expected %T", value.Val, expected)
	}

	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode lexicon record: %w", err)
	}
	return encoded, nil
}

// rawRecordJSON returns a record's JSON without requiring a known lexicon type,
// for records whose shape the caller decodes itself.
func rawRecordJSON(value *util.LexiconTypeDecoder) (json.RawMessage, error) {
	if value == nil || value.Val == nil {
		return nil, fmt.Errorf("Bobbin response is missing a record value")
	}

	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode lexicon record: %w", err)
	}
	return encoded, nil
}
