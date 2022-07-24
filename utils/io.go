package utils

import (
	"bufio"
	"fmt"
	"github.com/gogo/protobuf/proto"
)

func WriteProtoMsgToWriter(w *bufio.Writer, message proto.Message) error {
	marshal, err := proto.Marshal(message)
	if err != nil {
		return fmt.Errorf("数据序列化失败: %s", err.Error())
	}
	return WriteBytesToWriter(w, marshal)
}

func WriteBytesToWriter(w *bufio.Writer, data []byte) error {
	dataLen := int64(len(data))
	lenBytes, err := IntToBytes(dataLen)
	if err != nil {
		return fmt.Errorf("转换数据长度失败: %s", err.Error())
	}

	if _, err = w.Write(lenBytes); err != nil {
		return fmt.Errorf("数据长度写出失败: %s", err.Error())
	}

	if _, err = w.Write(data); err != nil {
		return fmt.Errorf("数据内容写出失败: %s", err.Error())
	}

	if err = w.Flush(); err != nil {
		return fmt.Errorf("数据通道缓存刷新失败: %s", err.Error())
	}

	return nil
}

func ReadProtoMsgByReader(r *bufio.Reader, res proto.Message) error {
	b, err := ReadBytesByReader(r)
	if err != nil {
		return err
	}

	if err = proto.Unmarshal(b, res); err != nil {
		return fmt.Errorf("反序列化数据内容失败: %s", err)
	}
	return err
}

func ReadBytesByReader(r *bufio.Reader) ([]byte, error) {
	var err error
	lenBuf := make([]byte, 8)
	for i := 0; i < 8; i++ {
		lenBuf[i], err = r.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("读取数据长度失败: %s", err.Error())
		}
	}

	l, err := BytesToInt[int64](lenBuf)
	if err != nil {
		return nil, fmt.Errorf("数据长度转换失败: %s", err.Error())
	}

	dataBuf := make([]byte, l)
	for i := int64(0); i < l; i++ {
		dataBuf[i], err = r.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("读取数据内容失败: %s", err.Error())
		}
	}
	return dataBuf, nil
}
