package utils

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestStringOrArrayImpl_UnmarshalJSON(t *testing.T) {
	var a *[]string
	var b []string
	assert.Nil(t, a)
	assert.Nil(t, b)

	a = &b
	assert.NotNil(t, a)
	assert.Nil(t, *a)

	c := struct {
		A *[]string
	}{}
	assert.Nil(t, c.A)
}

func TestStringOrArrayImpl_MarshalJSON(t *testing.T) {
	a := StringOrArrayImpl{
		strVal:  nil,
		arrVal:  nil,
		isArray: true,
	}
	assert.Equal(t, "", a.String())
	d, err := json.Marshal(&a)
	assert.NoError(t, err)
	assert.Equal(t, "null", string(d))
	b := StringOrArrayImpl{
		strVal:  nil,
		arrVal:  nil,
		isArray: false,
	}
	d, err = json.Marshal(&b)
	assert.NoError(t, err)
	assert.Equal(t, "null", string(d))

	c := StringOrArrayImpl{
		strVal:  PValue("test"),
		arrVal:  nil,
		isArray: false,
	}
	assert.False(t, c.IsArray())
	assert.Equal(t, []string{"test"}, c.AsArray())

	c = StringOrArrayImpl{
		strVal:  PValue(""),
		arrVal:  nil,
		isArray: false,
	}
	assert.False(t, c.IsArray())
	assert.Equal(t, []string{}, c.AsArray())

	v := []string{"test"}
	c = StringOrArrayImpl{
		strVal:  nil,
		arrVal:  &v,
		isArray: true,
	}
	assert.True(t, c.IsArray())
	assert.True(t, len(c.AsArray()) > 0)

	// nil test
	var e *StringOrArrayImpl
	assert.Equal(t, 0, len(e.AsArray()))

	testCases := []struct {
		Name         string
		TestJson     string
		ValueString  string
		MarshalValue string
		ShouldErr    bool
	}{
		{
			Name:         "字符串正确",
			TestJson:     `{"text":"test"}`,
			ValueString:  "test",
			MarshalValue: `{"text":"test"}`,
			ShouldErr:    false,
		},
		{
			Name:         "数组正确",
			TestJson:     `{"text":["testa","testb"]}`,
			ValueString:  "testa,testb",
			MarshalValue: `{"text":["testa","testb"]}`,
			ShouldErr:    false,
		},
		{
			Name:         "数组为空正确",
			TestJson:     `{"text":[]}`,
			ValueString:  "",
			MarshalValue: `{"text":[]}`,
			ShouldErr:    false,
		},
		{
			Name:         "数组为空正确",
			TestJson:     `{"text":null}`,
			ValueString:  "",
			MarshalValue: `{"text":null}`,
			ShouldErr:    false,
		},
		{
			Name:         "格式错误",
			TestJson:     `{"text":{"a":0}}`,
			ValueString:  "",
			MarshalValue: ``,
			ShouldErr:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			var a struct {
				Text StringOrArrayImpl `json:"text"`
			}
			if tc.ShouldErr {
				assert.Error(t, json.NewDecoder(strings.NewReader(tc.TestJson)).Decode(&a))
			} else {
				assert.NoError(t, json.NewDecoder(strings.NewReader(tc.TestJson)).Decode(&a))
				assert.Equal(t, tc.ValueString, a.Text.String())
				data, err := json.Marshal(&a)
				assert.NoError(t, err)
				assert.Equal(t, tc.MarshalValue, string(data))
			}
		})
	}

	//	// 字符串
	//	var a struct {
	//		Text StringOrArrayImpl `json:"text"`
	//	}
	//	assert.NoError(t, json.NewDecoder(strings.NewReader(`{
	//	"text": "test"
	//}`)).Decode(&a))
	//	assert.Equal(t, "test", a.Text.String())
	//	data, err := json.Marshal(&a)
	//	assert.NoError(t, err)
	//	assert.Equal(t, `{"text":"test"}`, string(data))
	//
	//	// 数组
	//	var b struct {
	//		Text StringOrArrayImpl `json:"text"`
	//	}
	//	assert.NoError(t, json.NewDecoder(strings.NewReader(`{
	//	"text": ["testa","testb"]
	//}`)).Decode(&b))
	//	assert.Equal(t, `testa,testb`, b.Text.String())
	//	data, err = json.Marshal(&b)
	//	assert.NoError(t, err)
	//	assert.Equal(t, `{"text":["testa","testb"]}`, string(data))
	//
	//	// object应该不能处理
	//	var c struct {
	//		Text StringOrArrayImpl `json:"text"`
	//	}
	//	assert.Error(t, json.NewDecoder(strings.NewReader(`{
	//	"text": {"a":1}
	//}`)).Decode(&c))
}

func TestNewStringOrArrayImpl(t *testing.T) {
	assert.Equal(t, "test", NewStringOrArrayImpl("test").String())
	var a []string = nil
	assert.Equal(t, "", NewStringOrArrayImpl(&a).String())
	assert.Equal(t, "", NewStringOrArrayImpl(nil).String())
	a = []string{"test1", "test2"}
	assert.Equal(t, "test1,test2", NewStringOrArrayImpl(a).String())
}
