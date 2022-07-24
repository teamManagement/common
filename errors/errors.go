package errors

import (
	"encoding/json"
	"fmt"
)

// ErrCode 错误代码
type ErrCode uint8

// Error 从错误代码构建Error结构
func (e ErrCode) Error(msg string) *Error {
	return &Error{
		Code: e,
		Msg:  msg,
	}
}

// Errorf 从错误代码构建Error结构, 内部msg使用 format 字符串格式
func (e ErrCode) Errorf(format string, args ...any) *Error {
	return e.Error(fmt.Sprintf(format, args...))
}

// Equal 判断错误是否相等
func (e ErrCode) Equal(err error) bool {
	targetErr, ok := ErrParse(err)
	if !ok {
		return ok
	}
	return targetErr.Equal(e)
}

// ErrParse 将一个异常尝试解析为一个 Error 类型
func ErrParse(err error) (*Error, bool) {
	switch t := err.(type) {
	case *Error:
		return t, true
	default:
		return nil, false
	}
}

// Error 具体的错误内容
type Error struct {
	// Code 错误代码
	Code ErrCode
	// Msg 错误消息
	Msg string
}

// Error 实现error接口
func (e *Error) Error() string {
	return e.Msg
}

func (e *Error) MarshalToJson() ([]byte, error) {
	if marshal, err := json.Marshal(e); err != nil {
		return nil, fmt.Errorf("序列化异常信息失败: %s", err.Error())
	} else {
		return marshal, nil
	}
}

// Equal 判断Error内的Code是否与预期匹配
func (e *Error) Equal(errCode ErrCode) bool {
	return errCode == e.Code
}

// UnmarshalJson 反序列化错误信息
func UnmarshalJson(data []byte) (*Error, error) {
	var err *Error
	if e := json.Unmarshal(data, &err); e != nil {
		return nil, fmt.Errorf("异常信息反序列化失败: %s", e.Error())
	}
	return err, nil
}

const (
	// ErrCodeUnknown 未定义的异常
	ErrCodeUnknown ErrCode = iota + 1
	// ErrCodeCmdNotFound 命令未识别
	ErrCodeCmdNotFound
)
