package utils

import (
	"encoding/json"
	"fmt"
	"strings"
)

// PValue 把一个值转换为pointer指针的格式，用与解决gorm默认值字段创建是赋值失败的问题
func PValue[T comparable](v T) *T {
	pv := v
	return &pv
}

// StringOrArrayImpl 用于实际存储数据，同时满足 String 或 Array 的接口
type StringOrArrayImpl struct {
	strVal  *string
	arrVal  *[]string
	isArray bool
}

// UnmarshalJSON 实现自定义的反序列化逻辑
func (s *StringOrArrayImpl) UnmarshalJSON(data []byte) error {
	var strVal string
	var arrVal []string

	// 有限处理数组，尝试解析为数组
	if err := json.Unmarshal(data, &arrVal); err == nil {
		s.arrVal = &arrVal
		s.isArray = true
		return nil
	}

	// 尝试解析为字符串
	if err := json.Unmarshal(data, &strVal); err == nil {
		if err := json.Unmarshal([]byte(strVal), &arrVal); err == nil {
			s.arrVal = &arrVal
			s.isArray = true
			return nil
		}

		s.strVal = &strVal
		s.isArray = false
		return nil
	}

	return fmt.Errorf("cannot unmarshal JSON as string or array: %s", data)
}

// MarshalJSON 实现自定义的序列化逻辑
func (s *StringOrArrayImpl) MarshalJSON() ([]byte, error) {
	if s.isArray {
		if s.arrVal != nil && *s.arrVal != nil {
			return json.Marshal(*s.arrVal)
		}
		return []byte("null"), nil
	}
	if s.strVal != nil {
		return json.Marshal(*s.strVal)
	}
	return []byte("null"), nil
}

func (s *StringOrArrayImpl) String() string {
	if s.strVal != nil {
		return *s.strVal
	} else {
		if s.arrVal == nil {
			return ""
		}
		return strings.Join(*s.arrVal, ",")
	}
}

func (s *StringOrArrayImpl) IsArray() bool {
	return s.isArray
}

func (s *StringOrArrayImpl) AsArray() []string {
	if s == nil {
		return []string{}
	}

	var arr []string
	if s.IsArray() {
		if v := s.arrVal; len(*v) > 0 {
			arr = *s.arrVal
		}
	} else if v := s.String(); v != "" {
		arr = []string{v}
	}
	if arr == nil {
		arr = []string{}
	}
	return arr
}

// NewStringOrArrayImpl 构造一个 StringOrArrayImpl 对象
func NewStringOrArrayImpl(v any) *StringOrArrayImpl {
	if v == nil {
		return &StringOrArrayImpl{PValue(""), nil, false}
	}
	switch i := v.(type) {
	case string:
		strVal := i
		return &StringOrArrayImpl{&strVal, nil, false}
	case []string:
		arrVal := i
		return &StringOrArrayImpl{nil, &arrVal, true}
	default:
		return &StringOrArrayImpl{PValue(""), nil, false}
	}
}
