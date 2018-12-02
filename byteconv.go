package main

import (
	"errors"
	"io"
)

func readInt16(r io.Reader) (i int16, err error) {
	tmp := make([]byte, 2)
	n, err := r.Read(tmp)

	if n != 2 {
		return -1, errors.New("readInt16: read too short")
	}

	return int16(tmp[0])<<8 | int16(tmp[1]), nil
}

func writeInt16(dest []byte, i int16) {
	if len(dest) != 2 {
		panic("writeInt16: destination size != 2")
	}

	dest[0] = byte(i >> 8)
	dest[1] = byte(i & 0xff)
}

func writeInt32(w io.Writer, i int) error {
	n, err := w.Write([]byte{
		byte(i >> 24),
		byte((i >> 16) & 0xff),
		byte((i >> 8) & 0xff),
		byte(i & 0xff)})
	if err != nil {
		return err
	}

	if n != 4 {
		return errors.New("writeInt32: write too short")
	}

	return nil
}

func writeString(dest []byte, str string) error {
	if len(dest) != 64 {
		return errors.New("writeString: dest length != 64")
	}

	bstr := []byte(str)

	if len(bstr) > 64 {
		return errors.New("writeString: string length > 64")
	}

	for i, c := range bstr {
		dest[i] = c
	}

	for i := len(bstr); i < 64; i++ {
		dest[i] = ' '
	}

	return nil
}
