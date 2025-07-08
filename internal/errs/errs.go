package errs

import "errors"

var ErrInsufficientFunds = errors.New("not enough balance")
var ErrUserNotFound = errors.New("user not found")
var ErrInvalidToken = errors.New("invalid token")
var ErrLoginAlreadyExists = errors.New("login already exists")
