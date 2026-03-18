package utils

import (
	"reflect"
)

type stream struct {
	data []interface{}
}

func NewStream(data interface{}) *stream {
	s := reflect.ValueOf(data)
	if s.Kind() != reflect.Slice {
		panic("not slice :(")
	}
	ret := make([]interface{}, s.Len())
	for i := 0; i < s.Len(); i++ {
		ret[i] = s.Index(i).Interface()
	}
	return &stream{
		data: ret,
	}

}

func NewTypedStream(data interface{}, dataType interface{}) *stream {
	s := reflect.ValueOf(data)
	if s.Kind() != reflect.Slice {
		panic("not slice :(")
	}
	ret := make([]interface{}, s.Len())
	for i := 0; i < s.Len(); i++ {
		ret[i] = s.Index(i).Interface()
	}

	if reflect.TypeOf(ret[0]) == reflect.TypeOf(dataType) {
		return &stream{data: ret}
	} else {
		panic("probably passing value instead of reference or vice versa")
	}
}

type boolFilter func(d interface{}) bool

func (s *stream) Filter(f boolFilter) *stream {
	a := make([]interface{}, 0)
	for _, d := range s.data {
		if f(d) {
			a = append(a, d)
		}
	}
	s.data = a
	return s
}

func (s *stream) Map(f func(d interface{}) interface{}) *stream {
	a := make([]interface{}, 0)
	for _, d := range s.data {
		a = append(a, f(d))
	}
	return NewStream(a)
}

func (s *stream) ForEach(f func(d interface{})) *stream {
	for _, d := range s.data {
		f(d)
	}
	return s
}

func (s *stream) Count() int {
	return len(s.data)
}

func (s *stream) Sum() int {
	sum := 0
	for _, d := range s.data {
		sum += d.(int)
	}
	return sum
}

func (s *stream) AnyMatch(f boolFilter) bool {
	for _, el := range s.data {
		if f(el) {
			return true
		}
	}
	return false
}

func (s *stream) FindFirst(f boolFilter) interface{} {
	for _, el := range s.data {
		if f(el) {
			return el
		}
	}
	return nil
}
