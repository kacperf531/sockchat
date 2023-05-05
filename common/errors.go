package common

import "errors"

var ErrUnauthorized = errors.New("unauthorized")
var ErrResourceConflict = errors.New("resource already exists")
