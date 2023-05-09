package common

import "errors"

var ErrNickAlreadyUsed = errors.New("this nick is already used")
var ErrInvalidRequest = errors.New("invalid request")
var ErrInternal = errors.New("internal error")
