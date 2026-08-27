package client

import "encoding/json"

// NullableString is a tri-state JSON field for PATCH bodies: unset (omitted
// from the JSON entirely, via `omitzero`), explicitly null (clears the
// column), or a value. The v1 PATCH routes treat "omitted" as "leave alone"
// and "null" as "clear", and a plain *string cannot express both.
type NullableString struct {
	Set   bool
	Value *string
}

// Null is an explicit JSON null.
func Null() NullableString { return NullableString{Set: true} }

// Str is an explicit value.
func Str(s string) NullableString { return NullableString{Set: true, Value: &s} }

// IsZero lets encoding/json's `omitzero` drop an unset field.
func (n NullableString) IsZero() bool { return !n.Set }

func (n NullableString) MarshalJSON() ([]byte, error) {
	if n.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(*n.Value)
}

func (n *NullableString) UnmarshalJSON(b []byte) error {
	n.Set = true
	if string(b) == "null" {
		n.Value = nil
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	n.Value = &s
	return nil
}
