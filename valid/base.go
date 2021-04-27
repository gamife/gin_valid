package valid

import (
	"reflect"
	"strconv"
	"time"
)

var (
	timeDurationType = reflect.TypeOf(time.Duration(0))
	timeType         = reflect.TypeOf(time.Time{})
)

// AsInt returns the parameter as a int64
// or panics if it can't convert
func AsInt(param string) int64 {
	i, err := strconv.ParseInt(param, 0, 64)
	PanicIf(err)

	return i
}

// AsIntFromTimeDuration parses param as time.Duration and returns it as int64
// or panics on error.
func AsIntFromTimeDuration(param string) int64 {
	d, err := time.ParseDuration(param)
	if err != nil {
		// attempt parsing as an an integer assuming nanosecond precision
		return AsInt(param)
	}
	return int64(d)
}

// AsIntFromType calls the proper function to parse param as int64,
// given a field's Type t.
func AsIntFromType(t reflect.Type, param string) int64 {
	switch t {
	case timeDurationType:
		return AsIntFromTimeDuration(param)
	default:
		return AsInt(param)
	}
}

// AsUint returns the parameter as a uint64
// or panics if it can't convert
func AsUint(param string) uint64 {

	i, err := strconv.ParseUint(param, 0, 64)
	PanicIf(err)

	return i
}

// AsFloat returns the parameter as a float64
// or panics if it can't convert
func AsFloat(param string) float64 {

	i, err := strconv.ParseFloat(param, 64)
	PanicIf(err)

	return i
}

// AsBool returns the parameter as a bool
// or panics if it can't convert
func AsBool(param string) bool {

	i, err := strconv.ParseBool(param)
	PanicIf(err)

	return i
}

func PanicIf(err error) {
	if err != nil {
		panic(err.Error())
	}
}
