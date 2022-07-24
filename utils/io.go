package utils

import (
	"bufio"
	"fmt"
)

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
