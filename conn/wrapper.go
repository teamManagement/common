package conn

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-base-lib/goextension"
	transportstream "github.com/go-base-lib/transport-stream"
	"net"
)

type MessageType uint

const (
	MessageTypeError MessageType = iota
	MessageTypeSuccess
)

type MessageInfo struct {
	Type    MessageType `json:"type,omitempty"`
	ErrCode uint        `json:"errCode,omitempty"`
	Message string      `json:"message,omitempty"`
	Data    []byte      `json:"data,omitempty"`
}

func NewErrorMessageInfo(msg string) *MessageInfo {
	return NewErrorMessageInfoWithCode(0, msg)
}

func NewErrorMessageInfoWithCode(code uint, msg string) *MessageInfo {
	return &MessageInfo{
		Type:    MessageTypeError,
		Message: msg,
		ErrCode: code,
	}
}

type Wrapper struct {
	rw  *bufio.ReadWriter
	err error
}

func NewWrapper(conn net.Conn) *Wrapper {
	return &Wrapper{
		rw: bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn)),
	}
}

func (w *Wrapper) Error() error {
	err := w.err
	w.err = nil
	return err
}

func (w *Wrapper) wrapperError(fn func() error) *Wrapper {
	if w.err != nil {
		return w
	}

	w.err = fn()
	return w
}

func (w *Wrapper) WriteByte(b byte) *Wrapper {
	return w.wrapperError(func() error {
		if err := w.rw.WriteByte(b); err != nil {
			return err
		}
		return w.rw.Flush()
	})
}

func (w *Wrapper) WriteErrMessage(msg string) *Wrapper {
	return w.WriteErrMessageWithCode(0, msg)
}

func (w *Wrapper) WriteErrMessageWithCode(errCode uint, msg string) *Wrapper {
	return w.WriteFormatJsonData(NewErrorMessageInfoWithCode(errCode, msg))
}

func (w *Wrapper) WriteFormatJsonData(data any) *Wrapper {
	if w.err != nil {
		return w
	}
	marshal, err := json.Marshal(data)
	if err != nil {
		w.err = err
		return w
	}

	return w.WriteFormatBytesData(marshal)
}

func (w *Wrapper) WriteFormatBytesData(data []byte) *Wrapper {
	marshal, _ := json.Marshal(&MessageInfo{
		Type: MessageTypeSuccess,
		Data: data,
	})

	return w.writeBytes(marshal)
}

func (w *Wrapper) writeBytes(data []byte) *Wrapper {
	return w.wrapperError(func() error {

		dataLen := len(data)
		lenBytes, err := transportstream.IntToBytes[int64](int64(dataLen))
		if err != nil {
			return err
		}

		if _, err = w.rw.Write(lenBytes); err != nil {
			return err
		}

		if _, err = w.rw.Write(data); err != nil {
			return err
		}
		return w.rw.Flush()
	})
}

func (w *Wrapper) ReadeFormatBytesData() (goextension.Bytes, error) {
	dataLenBytes, err := w.ReadeCountBytes(8)
	if err != nil {
		return nil, err
	}

	dataLen, err := transportstream.BytesToInt[int64](dataLenBytes)
	if err != nil {
		return nil, err
	}

	wrapperBytes, err := w.ReadeCountBytes(dataLen)
	if err != nil {
		return nil, err
	}

	var messageInfo *MessageInfo
	if err = json.Unmarshal(wrapperBytes, &messageInfo); err != nil {
		return nil, fmt.Errorf("数据格式解析失败: %s", err.Error())
	}

	if messageInfo.Type == MessageTypeSuccess {
		return messageInfo.Data, nil
	}

	return nil, errors.New(messageInfo.Message)
}

func (w *Wrapper) ReadeCountBytes(count int64) (goextension.Bytes, error) {
	buf := make([]byte, count)
	for i := int64(0); i < count; i++ {
		b, err := w.rw.ReadByte()
		if err != nil {
			return nil, err
		}
		buf[i] = b
	}
	return buf, nil
}

func (w *Wrapper) ReadFormatJsonData(res any) error {
	dataBytes, err := w.ReadeFormatBytesData()
	if err != nil {
		return err
	}

	return json.Unmarshal(dataBytes, &res)
}

func (w *Wrapper) ReadByte() (byte, error) {
	return w.rw.ReadByte()
}
