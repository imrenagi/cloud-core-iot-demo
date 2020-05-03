package errors

import "fmt"

var ErrInvalidArguments = fmt.Errorf("invalid arguments")
var ErrInternal = fmt.Errorf("internal error")
