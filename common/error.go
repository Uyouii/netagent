package common

import "fmt"

type ERROR int

type Error struct {
	Errno  ERROR  `json:"errno"`
	ErrMsg string `json:"errmsg"`
}

const (
	ERROR_OK ERROR = iota
	ERROR_EMPTY
	ERROR_INVALID_PARAMS
	ERROR_DISCONNECTED
)

var ERROR_MSG_MAP = map[ERROR]string{
	ERROR_OK:             "ok",
	ERROR_EMPTY:          "empty",
	ERROR_INVALID_PARAMS: "invalid params",
	ERROR_DISCONNECTED:   "disconnected",
}

func GetError(errno ERROR) error {
	return &Error{Errno: errno}
}

func GetErrorWithMsg(errno ERROR, msg string) error {
	return &Error{Errno: errno, ErrMsg: msg}
}

func (e *Error) Error() string {
	errMsg, ok := ERROR_MSG_MAP[e.Errno]
	if !ok {
		errMsg = "unknown error"
	}
	return fmt.Sprintf("errno: %v, %v, %v", e.Errno, errMsg, e.ErrMsg)
}

func IsError(err error, errno ERROR) bool {
	e, ok := err.(*Error)
	if !ok {
		return false
	}
	return e.Errno == errno
}
