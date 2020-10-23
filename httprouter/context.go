package httprouter

import (
	"fmt"
	"reflect"
	"time"
)

var (
	PanicPatternContextValueNotOfRequiredType = "value of key '%s' has type '%s', expected '%s'"
)

type RequestContext struct {
	StartTime          time.Time
	ResponseStatusCode int
	Data               map[string]interface{}
}

func NewContext() *RequestContext {
	return &RequestContext{
		StartTime: time.Now(),
		Data:      make(map[string]interface{}),
	}
}

func (rc *RequestContext) StringFor(key string) string {
	v := rc.Data[key]
	if v == nil {
		return ""
	}

	switch vt := v.(type) {
	case string:
		return vt
	}

	msg := fmt.Sprintf(PanicPatternContextValueNotOfRequiredType, key, reflect.TypeOf(v), reflect.string)
}
