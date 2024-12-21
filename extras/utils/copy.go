package utils

import (
	"errors"
	"io"
	"sync"
)

const DefaultBufferSize = 32 * 1024

var sharedBufferPool = sync.Pool{
	New: func() interface{} {
		buffer := make([]byte, DefaultBufferSize)
		return &buffer
	},
}

func getBuffer() *[]byte {
	buffer := sharedBufferPool.Get().(*[]byte)
	*buffer = (*buffer)[:cap(*buffer)]
	return buffer
}

func putBuffer(buffer *[]byte) {
	if buffer != nil {
		*buffer = (*buffer)[:cap(*buffer)]
		sharedBufferPool.Put(buffer)
	}
}

func CopyBuffer(dst io.Writer, src io.Reader) (written int64, err error) {
	buf := getBuffer()
	defer putBuffer(buf)
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
