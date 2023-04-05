package errors

import "errors"

var Unauthorized = errors.New("unauthorized")
var ResourceConflict = errors.New("resource already exists")
