package demo

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

var (
	testInput        = []string{"foo", "bar", "baz"}
	testOutput       = `["foo", "bar", "baz"]`
	testOutputLength = int64(len(testOutput))
)

func assertInt64(t *testing.T, expect, actual int64) {
	if actual != expect {
		t.Errorf("expected %d, got: %d", expect, actual)
	}
}

func assertString(t *testing.T, expect, actual string) {
	if actual != expect {
		t.Errorf("expected '%s', got: '%s'", expect, actual)
	}
}

func fatalOn(t *testing.T, err error) {
	if err == nil {
		return
	}
	t.Fatal(err)
}

func TestPathologicalCase(t *testing.T) {
	stringify := func(w io.Writer, a []string) (n int, err error) {
		w.Write([]byte{'['})
		for i := 0; i < len(a); i++ {
			if i > 0 {
				fmt.Fprint(w, ", ")
			}
			fmt.Fprintf(w, `"%s"`, a[i])
		}
		w.Write([]byte{']'})
		return
	}

	b := &bytes.Buffer{}
	_, err := stringify(b, testInput)
	fatalOn(t, err)
	assertString(t, testOutput, b.String())
}

func TestPedanticCase(t *testing.T) {
	stringify := func(w io.Writer, a []string) (n int64, err error) {
		var nn int

		// write opening bracket
		nn, err = w.Write([]byte{'['})
		if err != nil {
			return
		}
		n += int64(nn)

		for i := 0; i < len(a); i++ {
			if i > 0 {
				// write separator
				nn, err = fmt.Fprint(w, ", ")
				if err != nil {
					return
				}
				n += int64(nn)
			}

			// write quoted member
			nn, err = fmt.Fprintf(w, `"%s"`, a[i])
			if err != nil {
				return
			}
			n += int64(nn)
		}

		// write closing bracket
		nn, err = w.Write([]byte{']'})
		if err != nil {
			return
		}
		n += int64(nn)

		return
	}

	b := &bytes.Buffer{}
	n, err := stringify(b, testInput)
	fatalOn(t, err)
	assertInt64(t, testOutputLength, n)
	assertString(t, testOutput, b.String())
}

func TestAggregatorCase(t *testing.T) {
	stringify := func(w io.Writer, a []string) (n int64, err error) {
		w = NewAggregatedWriter(w)
		w.Write([]byte{'['})
		for i := 0; i < len(a); i++ {
			if i > 0 {
				fmt.Fprint(w, ", ")
			}
			fmt.Fprintf(w, `"%s"`, a[i])
		}
		w.Write([]byte{']'})
		return w.(*AggregatedWriter).Result()
	}

	b := &bytes.Buffer{}
	n, err := stringify(b, testInput)
	fatalOn(t, err)
	assertInt64(t, testOutputLength, n)
	assertString(t, testOutput, b.String())
}
