package demo

import "io"

type AggregatedWriter struct {
	w   io.Writer
	n   int64
	err error
}

func NewAggregatedWriter(w io.Writer) *AggregatedWriter {
	if ag, ok := w.(*AggregatedWriter); ok {
		return ag
	}
	return &AggregatedWriter{w: w}
}

func (w *AggregatedWriter) Write(p []byte) (n int, err error) {
	if w.err != nil {
		return 0, w.err
	}
	n, err = w.w.Write(p)
	w.n += int64(n)
	w.err = err
	return
}

func (w *AggregatedWriter) N() int64                     { return w.n }
func (w *AggregatedWriter) Err() error                   { return w.err }
func (w *AggregatedWriter) Result() (n int64, err error) { return w.n, w.err }
