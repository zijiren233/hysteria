package server

import (
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/apernet/hysteria/core/v2/internal/utils"
)

const (
	trafficReportInterval  = 10 * time.Second
	trafficReportThreshold = 1024 * 1024 * 1024 // 10MB
)

var errDisconnect = errors.New("traffic logger requested disconnect")

func copyBufferLog(dst io.Writer, src io.Reader, log func(n uint64) bool) error {
	bufP := utils.GetBuffer()
	defer utils.PutBuffer(bufP)
	buf := *bufP
	var unreported int
	lastReportTime := time.Now()

	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			unreported += nr

			// Check if we should report traffic
			shouldReport := unreported >= trafficReportThreshold ||
				time.Since(lastReportTime) >= trafficReportInterval

			if shouldReport && unreported > 0 {
				if !log(uint64(unreported)) {
					// Log returns false, which means that the client should be disconnected
					return errDisconnect
				}
				unreported = 0
				lastReportTime = time.Now()
			}

			_, ew := dst.Write(buf[0:nr])
			if ew != nil {
				return ew
			}
		}
		if er != nil {
			// Report any remaining unreported traffic before returning
			if unreported > 0 {
				log(uint64(unreported))
			}
			if er == io.EOF {
				// EOF should not be considered as an error
				return nil
			}
			return er
		}
	}
}

func copyTwoWayEx(
	id string,
	serverRw, remoteRw io.ReadWriter,
	l TrafficLogger,
	stats *StreamStats,
) error {
	var wg sync.WaitGroup
	var err atomic.Value

	wg.Add(1)
	go func() {
		defer wg.Done()
		e := copyBufferLog(serverRw, remoteRw, func(n uint64) bool {
			stats.LastActiveTime.Store(time.Now())
			stats.Rx.Add(n)
			return l.LogTraffic(id, 0, n)
		})
		if e != nil {
			err.CompareAndSwap(nil, e)
		}
	}()

	e := copyBufferLog(remoteRw, serverRw, func(n uint64) bool {
		stats.LastActiveTime.Store(time.Now())
		stats.Tx.Add(n)
		return l.LogTraffic(id, n, 0)
	})
	if e != nil {
		err.CompareAndSwap(nil, e)
	}

	wg.Wait()

	e, _ = err.Load().(error)
	return e
}

// copyTwoWay is the "fast-path" version of copyTwoWayEx that does not log traffic or update stream stats.
// It uses the built-in io.Copy instead of our own copyBufferLog.
func copyTwoWay(serverRw, remoteRw io.ReadWriter) error {
	var wg sync.WaitGroup
	var err atomic.Value

	wg.Add(1)
	go func() {
		defer wg.Done()
		_, e := utils.CopyBuffer(serverRw, remoteRw)
		if e != nil {
			err.CompareAndSwap(nil, e)
		}
	}()

	_, e := utils.CopyBuffer(remoteRw, serverRw)
	if e != nil {
		err.CompareAndSwap(nil, e)
	}

	wg.Wait()

	e, _ = err.Load().(error)
	return e
}
