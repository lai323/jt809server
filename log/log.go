package log

import (
	"bytes"
	"io"
	"os"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/go-logfmt/logfmt"
)

type colorBuffer struct {
	bytes.Buffer
}

type logdevEncoder struct {
	*logfmt.Encoder
	buf bytes.Buffer
}

func (enc *logdevEncoder) EncodeStack(stack interface{}) (bool, error) {
	switch v := stack.(type) {
	case string:
		_, err := io.WriteString(&enc.buf, "\n"+v)
		return true, err
	case []byte:
		_, err := enc.buf.Write(append([]byte("\n"), v...))
		return true, err
	}
	return false, nil
}

func (enc *logdevEncoder) EncodeKeyval(k, v interface{}) error {
	err := enc.Encoder.EncodeKeyval(k, v)
	if err == logfmt.ErrUnsupportedKeyType {
		return nil
	}
	if _, ok := err.(*logfmt.MarshalerError); ok || err == logfmt.ErrUnsupportedValueType {
		v = err
		err = enc.Encoder.EncodeKeyval(k, v)
	}
	if err != nil {
		return err
	}
	return nil
}

// EncodeKeyvals writes the logfmt encoding of keyvals to the stream. Keyvals
// is a variadic sequence of alternating keys and values. Keys of unsupported
// type are skipped along with their corresponding value. Values of
// unsupported type or that cause a MarshalerError are replaced by their error
// but do not cause EncodeKeyvals to return an error. If a non-nil error is
// returned some key/value pairs may not have be written.
func (enc *logdevEncoder) EncodeKeyvals(keyvals ...interface{}) error {
	if len(keyvals) == 0 {
		return nil
	}
	if len(keyvals)%2 == 1 {
		keyvals = append(keyvals, nil)
	}
	var stack interface{}
	for i := 0; i < len(keyvals); i += 2 {
		k, v := keyvals[i], keyvals[i+1]

		if kstr, ok := k.(string); ok {
			if kstr == "stack" {
				stack = v
				continue
			}
		}

		err := enc.EncodeKeyval(k, v)
		if err != nil {
			return err
		}
	}

	if stack != nil {
		ok, err := enc.EncodeStack(stack)
		if err != nil {
			return err
		}
		if !ok {
			err := enc.EncodeKeyval("stack", stack)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (l *logdevEncoder) Reset() {
	l.Encoder.Reset()
	l.buf.Reset()
}

var logdevEncoderPool = sync.Pool{
	New: func() interface{} {
		var enc logdevEncoder
		enc.Encoder = logfmt.NewEncoder(&enc.buf)
		return &enc
	},
}

type logdevLogger struct {
	w io.Writer
}

// NewLogdevLogger returns a logger that encodes keyvals to the Writer in
// logdev format. Each log event produces no more than one call to w.Write.
// The passed Writer must be safe for concurrent use by multiple goroutines if
// the returned Logger will be used concurrently.
// Logdev format does not escape the newline characters in the "stack" field
func NewLogdevLogger(w io.Writer) log.Logger {
	return &logdevLogger{w}
}

func NewLogdevStdoutLogger(logLevel level.Option) log.Logger {
	var logger log.Logger
	logger = &logdevLogger{log.NewSyncWriter(os.Stdout)}
	logger = level.NewFilter(logger, logLevel)
	logger = log.With(logger, "ts", DevTimestamp)
	logger = log.With(logger, "caller", log.DefaultCaller)
	return logger
}

func (l logdevLogger) Log(keyvals ...interface{}) error {
	enc := logdevEncoderPool.Get().(*logdevEncoder)
	enc.Reset()
	defer logdevEncoderPool.Put(enc)

	if err := enc.EncodeKeyvals(keyvals...); err != nil {
		return err
	}

	// Add newline to the end of the buffer
	if err := enc.EndRecord(); err != nil {
		return err
	}

	// The Logger interface requires implementations to be safe for concurrent
	// use by multiple goroutines. For this implementation that means making
	// only one call to l.w.Write() for each call to Log.
	if _, err := l.w.Write(enc.buf.Bytes()); err != nil {
		return err
	}
	return nil
}

var execTime = time.Now()
var DevTimestamp log.Valuer = func() interface{} {
	return int64(time.Now().Sub(execTime) / time.Second)
}
var DevFullTimestamp = log.TimestampFormat(time.Now, "2006-01-02 15:04:05")
