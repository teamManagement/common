package conn

import (
	"bufio"
	"encoding/json"
	"github.com/go-base-lib/goextension"
	transportstream "github.com/go-base-lib/transport-stream"
	"net"
)

type MessageType uint

const (
	MessageTypeError MessageType = iota
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
	rw *bufio.ReadWriter
}

func NewWrapper(conn net.Conn) *Wrapper {
	return &Wrapper{
		rw: bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn)),
	}
}

func (w *Wrapper) WriteErrMessage(msg string) error {
	return w.writeErrMessageWithCode(0, msg)
}

func (w *Wrapper) writeErrMessageWithCode(errCode uint, msg string) error {
	return w.WriteFormatJsonData(NewErrorMessageInfoWithCode(errCode, msg))
}

func (w *Wrapper) WriteFormatJsonData(data any) error {
	marshal, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return w.WriteFormatBytesData(marshal)
}

func (w *Wrapper) WriteFormatBytesData(data []byte) error {
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

	return w.ReadeCountBytes(dataLen)
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
