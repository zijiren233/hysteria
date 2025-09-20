package utils

import (
	"errors"
	"io"
	"sync"
)

const DefaultBufferSize = 32 * 1024

var sharedBufferPool = sync.Pool{
	New: func() any {
		buffer := make([]byte, DefaultBufferSize)
		return &buffer
	},
}

func GetBuffer() *[]byte {
	buffer := sharedBufferPool.Get().(*[]byte)
	*buffer = (*buffer)[:cap(*buffer)]
	return buffer
}

func PutBuffer(buffer *[]byte) {
	if buffer != nil {
		*buffer = (*buffer)[:cap(*buffer)]
		sharedBufferPool.Put(buffer)
	}
}

func CopyBuffer(dst io.Writer, src io.Reader) (written int64, err error) {
	buf := GetBuffer()
	defer PutBuffer(buf)
	for {
		nr, er := src.Read(*buf)
		if nr > 0 {
			nw, ew := dst.Write((*buf)[0:nr])
			if nw < 0 || nr < nw {
				nw = 0
				if ew == nil {
					ew = errors.New("invalid write result")
				}
			}
			written += int64(nw)
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return written, err
}

func CopyBufferN(dst io.Writer, src io.Reader, n int64) (written int64, err error) {
	written, err = CopyBuffer(dst, io.LimitReader(src, n))
	if written == n {
		return n, nil
	}
	if written < n && err == nil {
		// src stopped early; must have been EOF.
		err = io.EOF
	}
	return
}
