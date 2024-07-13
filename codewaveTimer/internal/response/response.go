package response

import (
	"codewave-timer/codewaveTimer/internal/types"
)

const (
	SuccessCode = 200
	FailCode    = 400
)

func Success(data string) *types.Response {
	return newResponse(SuccessCode, "success", data)
}

func Fail(data string) *types.Response {
	return newResponse(FailCode, "fail", data)
}

func newResponse(code int, message string, data string) *types.Response {
	return &types.Response{
		Code:    int32(code),
		Message: message,
		Data:    data,
	}
}
