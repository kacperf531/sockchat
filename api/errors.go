package api

import "errors"

var (
	ErrInvalidRequest = errors.New("invalid request")
	ErrInternal       = errors.New("internal error")
	ErrUnauthorized   = errors.New("unauthorized")

	ErrBasicTokenRequired    = errors.New("basic token is required")
	ErrCouldNotDecodeToken   = errors.New("could not decode provided token")
	ErrMetadataNotProvided   = errors.New("metadata not provided")
	ErrAuthorizationRequired = errors.New("authorization header is required")

	ErrNickAlreadyUsed       = errors.New("this nick is already used")
	ErrNickRequired          = errors.New("nick is required")
	ErrPasswordRequired      = errors.New("password is required")
	ErrUserNotFound          = errors.New("user not found")
	ErrChannelNotFound       = errors.New("channel not found")
	ErrChannelDoesNotExist   = errors.New("channel does not exist")
	ErrChannelAlreadyExists  = errors.New("channel with this name already exists")
	ErrUserNotInChannel      = errors.New("user is not member of this channel")
	ErrUserAlreadyInChannel  = errors.New("user is already member of this channel")
	ErrEmptyChannelName      = errors.New("channel's `name` is required")
	ErrMessageNotSent        = errors.New("message could not be sent")
	ErrInvalidGroupBy        = errors.New("invalid `group_by` value. Must be one of: day, hour, minute")
	ErrInvalidDateFormat     = errors.New("invalid date format. Must be: `YYYY-MM-DD HH:mm")
	ErrFromMissing           = errors.New("`from` is required")
	ErrToMissing             = errors.New("`to` is required")
	ErrInvalidRange          = errors.New("invalid range. `from` must be before `to`")
	ErrMaxReportSizeExceeded = errors.New("max report size exceeded")
)
