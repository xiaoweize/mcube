package queryparams_test

import (
	"net/url"
	"reflect"
	"testing"

	"github.com/infraboard/mcube/v2/http/queryparams"
)

type namedString string
type namedBool bool

type bar struct {
	Float1   float32 `json:"float1"`
	Float2   float64 `json:"float2"`
	Int1     int64   `json:"int1,omitempty"`
	Int2     int32   `json:"int2,omitempty"`
	Int3     int16   `json:"int3,omitempty"`
	Str1     string  `json:"str1,omitempty"`
	Ignored  int
	Ignored2 string
}

type foo struct {
	Str       string            `json:"str"`
	Integer   int               `json:"integer,omitempty"`
	Slice     []string          `json:"slice,omitempty"`
	Boolean   bool              `json:"boolean,omitempty"`
	NamedStr  namedString       `json:"namedStr,omitempty"`
	NamedBool namedBool         `json:"namedBool,omitempty"`
	Foobar    bar               `json:"foobar,omitempty"`
	Testmap   map[string]string `json:"testmap,omitempty"`
}

type baz struct {
	Ptr  *int  `json:"ptr"`
	Bptr *bool `json:"bptr,omitempty"`
}

// childStructs tests some of the types we serialize to query params for log API calls
// notably, the nested time struct
type childStructs struct {
	Container    string `json:"container,omitempty"`
	Follow       bool   `json:"follow,omitempty"`
	Previous     bool   `json:"previous,omitempty"`
	SinceSeconds *int64 `json:"sinceSeconds,omitempty"`
	TailLines    *int64 `json:"tailLines,omitempty"`
}

func validateResult(t *testing.T, input interface{}, actual, expected url.Values) {
	local := url.Values{}
	for k, v := range expected {
		local[k] = v
	}
	for k, v := range actual {
		if ev, ok := local[k]; !ok || !reflect.DeepEqual(ev, v) {
			if !ok {
				t.Errorf("%#v: actual value key %s not found in expected map", input, k)
			} else {
				t.Errorf("%#v: values don't match: actual: %#v, expected: %#v", input, v, ev)
			}
		}
		delete(local, k)
	}
	if len(local) > 0 {
		t.Errorf("%#v: expected map has keys that were not found in actual map: %#v", input, local)
	}
}

func TestConvert(t *testing.T) {
	sinceSeconds := int64(123)
	tailLines := int64(0)

	tests := []struct {
		input    interface{}
		expected url.Values
	}{
		{
			input: &foo{
				Str: "hello",
			},
			expected: url.Values{"str": {"hello"}},
		},
		{
			input: &foo{
				Str:     "test string",
				Slice:   []string{"one", "two", "three"},
				Integer: 234,
				Boolean: true,
			},
			expected: url.Values{"str": {"test string"}, "slice": {"one", "two", "three"}, "integer": {"234"}, "boolean": {"true"}},
		},
		{
			input: &foo{
				Str:       "named types",
				NamedStr:  "value1",
				NamedBool: true,
			},
			expected: url.Values{"str": {"named types"}, "namedStr": {"value1"}, "namedBool": {"true"}},
		},
		{
			input: &foo{
				Str: "don't ignore embedded struct",
				Foobar: bar{
					Float1: 5.0,
				},
			},
			expected: url.Values{"str": {"don't ignore embedded struct"}, "float1": {"5"}, "float2": {"0"}},
		},
		{
			// Ignore untagged fields
			input: &bar{
				Float1:   23.5,
				Float2:   100.7,
				Int1:     1,
				Int2:     2,
				Int3:     3,
				Ignored:  1,
				Ignored2: "ignored",
			},
			expected: url.Values{"float1": {"23.5"}, "float2": {"100.7"}, "int1": {"1"}, "int2": {"2"}, "int3": {"3"}},
		},
		{
			// include fields that are not tagged omitempty
			input: &foo{
				NamedStr: "named str",
			},
			expected: url.Values{"str": {""}, "namedStr": {"named str"}},
		},
		{
			input: &baz{
				Ptr:  intp(5),
				Bptr: boolp(true),
			},
			expected: url.Values{"ptr": {"5"}, "bptr": {"true"}},
		},
		{
			input: &baz{
				Bptr: boolp(true),
			},
			expected: url.Values{"ptr": {""}, "bptr": {"true"}},
		},
		{
			input: &baz{
				Ptr: intp(5),
			},
			expected: url.Values{"ptr": {"5"}},
		},
		{
			input: &childStructs{
				Container:    "mycontainer",
				Follow:       true,
				Previous:     true,
				SinceSeconds: &sinceSeconds,
				TailLines:    nil,
			},
			expected: url.Values{"container": {"mycontainer"}, "follow": {"true"}, "previous": {"true"}, "sinceSeconds": {"123"}},
		},
		{
			input: &childStructs{
				Container:    "mycontainer",
				Follow:       true,
				Previous:     true,
				SinceSeconds: &sinceSeconds,
				TailLines:    &tailLines,
			},
			expected: url.Values{"container": {"mycontainer"}, "follow": {"true"}, "previous": {"true"}, "sinceSeconds": {"123"}, "tailLines": {"0"}},
		},
	}

	for _, test := range tests {
		result, err := queryparams.Convert(test.input)
		if err != nil {
			t.Errorf("Unexpected error while converting %#v: %v", test.input, err)
		}
		validateResult(t, test.input, result, test.expected)
	}
}

func intp(n int) *int { return &n }

func boolp(b bool) *bool { return &b }

type Address struct {
	Street string `json:"street"`
	City   string `json:"city"`
}

type Person struct {
	Name  string `json:"name"`
	Age   int    `json:"age"`
	Email string `json:"email"`
	Address
}

func TestParamConvert(t *testing.T) {
	p := Person{
		Name:  "xxx",
		Age:   10,
		Email: "xxxxx",
		Address: Address{
			Street: "sdfsf",
			City:   "ccsdfdf",
		},
	}
	v, err := queryparams.Convert(p)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(v)
}
