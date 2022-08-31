package cmd

import (
	"encoding/json"
	"fmt"
	transportstream "github.com/go-base-lib/transport-stream"
	"github.com/gogo/protobuf/proto"
	"github.com/teamManagement/common/errors"
	"io"
	"net"
	"strings"
)

type ExchangeData []byte

func (e ExchangeData) UnmarshalJson(i any) error {
	if e == nil {
		return nil
	}

	if err := json.Unmarshal(e, i); err != nil {
		return fmt.Errorf("数据尝试从JSON反序列化到结构体失败: %s", err.Error())
	}

	return nil
}

func (e ExchangeData) UnmarshalProto(msg proto.Message) error {
	if e == nil {
		return nil
	}

	if err := proto.Unmarshal(e, msg); err != nil {
		return fmt.Errorf("数据尝试从proto反序列化到结构体失败: %s", err.Error())
	}
	return nil
}

func NewExchangeDataByStr(str string) ExchangeData {
	return []byte(str)
}

func NewExchangeDataByJson(data any) (ExchangeData, error) {
	marshal, err := json.Marshal(data)
	return marshal, err
}

func NewExchangeDataByJsonMust(data any) ExchangeData {
	marshal, _ := json.Marshal(data)
	return marshal
}

func NewExchangeDataByProto(data proto.Message) (ExchangeData, error) {
	marshal, err := proto.Marshal(data)
	return marshal, err
}

func NewExchangeDataByProtoMust(data proto.Message) ExchangeData {
	marshal, _ := proto.Marshal(data)
	return marshal
}

type Handler func(stream *transportstream.Stream, conn net.Conn) (ExchangeData, error)

var cmdMap = map[Name]Handler{}

func Route(stream *transportstream.Stream, conn net.Conn) error {
	sendEndOk := false
	defer func() {
		if sendEndOk {
			return
		}
		_ = stream.WriteEndMsg()
		for {
			if _, err := stream.ReceiveMsg(); err == transportstream.StreamIsEnd || err == io.EOF || strings.Contains(err.Error(), "connection reset by peer") {
				return
			}
		}
	}()
	defer func() {
		e := recover()
		if e != nil {
			errMsg := ""
			switch r := e.(type) {
			case string:
				errMsg = r
			case error:
				errMsg = r.Error()
			}
			_ = stream.WriteError(errors.ErrCodeUnknown.Newf("未知的指令处理异常: %s", errMsg))
		}
	}()

	cmdBytes, err := stream.ReceiveMsg()
	if err != nil {
		_ = stream.WriteError(errors.ErrCodeReadCommand.New("读取命令码失败: " + err.Error()))
		return err
	}

	cmdName := Name(cmdBytes)
	cmdHandle, ok := cmdMap[cmdName]
	if !ok {
		_ = stream.WriteError(errors.ErrCodeCommandUndefined.Newf("命令[%s]未被识别", cmdName))
		return nil
	}

	if err = stream.WriteMsg(nil, transportstream.MsgFlagSuccess); err != nil {
		return err
	}

	if nextData, err := cmdHandle(stream, conn); err != nil {
		if err == transportstream.StreamIsEnd {
			return nil
		}
		switch e := err.(type) {
		case *transportstream.ErrInfo:
			_ = stream.WriteError(e)
		default:
			_ = stream.WriteError(errors.ErrCodeUnknown.New(err.Error()))
		}
		return nil
	} else {
		if err = stream.WriteEndMsgWithData(nextData); err != nil {
			return nil
		}
		sendEndOk = true

		for {
			if _, err = stream.ReceiveMsg(); err == transportstream.StreamIsEnd {
				return nil
			}
		}

	}

}

type RWStreamInterface interface {
	WriteStreamInterface
	ReadLine() ([]byte, bool, error)
}

type WriteStreamInterface interface {
	Write([]byte) (int, error)

	Flush() error
}

type ExchangeOption struct {
	// StreamHandle 流拦截器
	StreamHandle func(exchangeData ExchangeData, stream *transportstream.Stream) (ExchangeData, error)
	// StreamErrHandle 处理除了 transportstream.StreamIsEnd 与 io.EOF 之外的所有异常
	// breakStream 表示是否中断流, 如果返回true则向对断发送 transportstream.StreamIsEnd 指令并跳出流监听
	// targetErr 将把转换之后的异常信息发送至服务器端
	StreamErrHandle func(exchangeData ExchangeData, err error) (breakStream bool, targetErr *transportstream.ErrInfo)
	// Data 要发送的数据
	Data any
}

type Name string

func emptyStreamHandler(exchangeData ExchangeData, stream *transportstream.Stream) (ExchangeData, error) {
	return nil, transportstream.StreamIsEnd
}

// SendCommand 发送一条命令到对端
func (c Name) SendCommand(stream *transportstream.Stream) error {
	if err := stream.WriteMsg([]byte(c), transportstream.MsgFlagSuccess); err != nil {
		return err
	}

	if _, err := stream.ReceiveMsg(); err != nil {
		return err
	}
	return nil
}

// ExchangeWithOption 交换数据到对端，数据为一来一回
func (c Name) ExchangeWithOption(stream *transportstream.Stream, option *ExchangeOption) (ExchangeData, error) {
	defer stream.WriteEndMsg()

	if option.StreamHandle == nil {
		option.StreamHandle = emptyStreamHandler
	}

	if err := c.SendCommand(stream); err != nil {
		return nil, err
	}

	if option.Data != nil {
		if err := stream.WriteJsonMsg(option.Data); err != nil {
			return nil, err
		}
	} else {
		if err := stream.WriteMsg(nil, transportstream.MsgFlagSuccess); err != nil {
			return nil, err
		}
	}

	for {
		msg, err := stream.ReceiveMsg()
		if err == transportstream.StreamIsEnd {
			return msg, nil
		}

		if err != nil {
			if err != transportstream.StreamIsEnd && option.StreamErrHandle != nil {
				breakStream, e := option.StreamErrHandle(msg, err)
				if e != nil {
					_ = stream.WriteError(e)
				}

				if breakStream {
					return msg, e
				}
				continue
			}
			for {
				if _, e := stream.ReceiveMsg(); e == transportstream.StreamIsEnd || e == io.EOF || strings.Contains(err.Error(), "connection reset by peer") {
					break
				}
			}
			return msg, err
		}

		nextData, err := option.StreamHandle(msg, stream)
		if err == transportstream.StreamIsEnd {
			if err = stream.WriteEndMsgWithData(nextData); err != nil {
				return nil, fmt.Errorf("接收结束消息失败: %s", err.Error())
			}
			return nil, nil
		}

		if err != nil {
			switch e := err.(type) {
			case *transportstream.ErrInfo:
				if err = stream.WriteError(e); err != nil {
					return nil, err
				}
			default:
				_err := errors.ErrCodeUnknown.New(err.Error())
				_err.RawData = nextData
				if err = stream.WriteError(_err); err != nil {
					return nil, err
				}
			}
			continue
		}
		if nextData != nil {
			if err = stream.WriteMsg(nextData, transportstream.MsgFlagSuccess); err != nil {
				return nil, err
			}
		}
	}

}

func (c Name) ExchangeWithData(data any, stream *transportstream.Stream) (ExchangeData, error) {
	return c.ExchangeWithOption(stream, &ExchangeOption{
		Data: data,
	})
}

func (c Name) Exchange(stream *transportstream.Stream) (ExchangeData, error) {
	return c.ExchangeWithData(nil, stream)
}

func (c Name) Registry(handle Handler) {
	cmdMap[c] = handle
}

const (
	// Login 登录
	Login = "/login"
	// Registry 注册
	Registry = "/registry"
	// Forgot 忘记密码
	Forgot = "/forgot"
)
