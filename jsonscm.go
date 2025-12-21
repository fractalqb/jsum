package jsum

type jscmType struct {
	Type string `json:"type"`
}

type jscmNumber struct {
	jscmType
	Min *float64 `json:"minimum,omitempty"`
	Max *float64 `json:"maximum,omitempty"`
}

type jscmString struct {
	jscmType
	Format string `json:"format,omitempty"`
	MinLen *int   `json:"minLength,omitempty"`
	MaxLen *int   `json:"maxLength,omitempty"`
}

type jscmArray struct {
	jscmType
	MinItems int `json:"minItems"`
	MaxItems int `json:"maxItems"`
	Items    any `json:"items,omitempty"`
}

type jscmOneOf struct {
	OneOf []any `json:"oneOf"`
}

type jscmObj struct {
	jscmType
	Required []string       `json:"required,omitempty"`
	Props    map[string]any `json:"properties"`
}
