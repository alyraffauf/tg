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
