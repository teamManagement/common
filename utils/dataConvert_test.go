package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestIntToBytes(t *testing.T) {
	a := assert.New(t)

	testData := []struct {
		num any
		bit int
	}{
		{int8(1), 1},
		{int16(1), 2},
		{int32(1), 4},
		{int64(1), 8},
		{uint8(1), 1},
		{uint16(1), 2},
		{uint32(1), 4},
		{uint64(1), 8},
	}

	for _, data := range testData {
		var (
			b   []byte
			err error
		)
		switch d := data.num.(type) {
		case uint8:
			b, err = IntToBytes(d)
		case uint16:
			b, err = IntToBytes(d)
		case uint32:
			b, err = IntToBytes(d)
		case uint64:
			b, err = IntToBytes(d)
		case int8:
			b, err = IntToBytes(d)
		case int16:
			b, err = IntToBytes(d)
		case int32:
			b, err = IntToBytes(d)
		case int64:
			b, err = IntToBytes(d)
		default:
			t.Error("未知的数值")
		}
		if !a.NoError(err) {
			return
		}

		a.Equal(len(b), data.bit)
	}

}

func TestBytesToInt(t *testing.T) {
	a := assert.New(t)
	bytes, err := IntToBytes(int64(16))
	a.NoError(err)

	r, err := BytesToInt[int64](bytes)
	a.NoError(err)

	a.Equal(r, int64(16))

	b, err := IntToBytes(r)
	a.NoError(err)

	a.Equal(len(b), 8)

}
