# One less `if err != nil { return err }`

Go is famous for its verbose error handling (only because it coerces you to
handle every case) but there are some patterns to keep it under control.

This isn't meant to be a _Ten Commandments of Error Handling_ or even a good
idea; it's just a pattern than has been useful to me.

A common scenario for poor error handling in practice is when using `io`
operations. Specifically, _writes_ typically blow out the length of your method
with repetitive error checking or they simply go without error handling at all.

We've all see this code:

```go
package main

import "fmt"

func main() {
    fmt.Println("hello world")
}
```

What's the one thing missing? Error handling. Yes, I concede, it's not
exactly necessary when writing to `os.Stdout` (this is what happens under the
hood in `fmt.Println`). What if we're writing to a file and we do need to make
sure it succeeds?

Easy. Let's check for errors when we write to the file:

```go
package main

import "fmt"

func main() {
    // create a file and check for errors
    f, err := os.Create("./helloWorld.txt")
    if err != nil {
        log.Fatal(err)
    }
    defer f.Close()

    // write to file and check for errors
    _, err = fmt.Fprintln("hello world")
    if err != nil {
        log.Fatal(err)
    }
}
```

Cool beans. What if the content we're writing is a little more sophisticated and
requires multiple successive writes? Maybe we're implementing our own JSON
stringifier.

Let's implement a naive stringifier that takes a list of strings and prints it
to a writer as a JSON array.

```go
func Stringify(w io.Writer, a []string) (n int, err error) {
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
```

Now, we get a lovely JSON array:

```go
func main() {
    Stringify(os.Stdout, []string{"hello", "world"})
}
```

```json
["hello", "world"]
```

This will work. But `Stringify` always returns `0, nil`, no matter what
happens. We need some bookkeeping around each write: 

```go
func Stringify(w io.Writer, a []string) (n int, err error) {
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
```

Oof, that's verbose! We're only doing four writes. You can see why so many
gophers might opt for ignoring the occasional cheaky error.

There is a more elegant way to solve this that leverages the power of Go's
`io.Writer` interface. Let's create a middleware writer that wraps our output
writer, keeps track of the number of bytes written and handles any errors along
the way.

Because naming is hard, we'll call it the `AggregatedWriter`:

```go
type AggregatedWriter struct {
    w   io.Writer // underlying writer
    n   int64     // cumulative total of bytes written to w
    err error     // any error returned by w
}

func NewAggregatedWriter(w io.Writer) *AggregatedWriter {
    if ag, ok := w.(*AggregatedWriter); ok {
        return ag // w was already an AggregatedWriter
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
```

The implementation is very simple but we hopefully get a lot of miles out of it.

The `AggregatedWriter` implements `io.Writer` so we can use it anywhere you
write to a file or buffer. Let's use it in our `Stringify` method and then talk
about how it works.

```go
func Stringify(w io.Writer, a []string) (n int, err error) {
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
```

It's not much longer that our first example, only this time the caller gets a
valid byte count and error result!

Looking into our `Stringify` method, it first wraps the given writer in an
`AggregatedWriter`, performs writes as before (with no error checking) and then
returns a result from the `AggregatedWriter` which will be the cumulative byte
count of all writes and any errors that occured writing to `os.Stdout`.

Let's take a closer look at the `AggregatedWriter`:

```go
// Write implements the io.Writer interface.
func (w *AggregatedWriter) Write(p []byte) (n int, err error) {
    // check if an error occurred previously - if so, abort this write and
    // return the previous error
    if w.err != nil {
        return 0, w.err
    }

    // write to the underlying writer and copy the bytes written and any
    // error to our return values
    n, err = w.w.Write(p)

    // increment the total written across all calls to this method
    w.n += int64(n)

    // store any error
    w.err = err

    // return (n, err) declared in the func signature
    return
}
```

There is a trade-off to this approach that you may need to pay attention to: 
execution is not interrupted when a write fails. The right way to use this is to
do all your writes, check that they worked, then do stuff that depends on it. If
that's not feasible, this approach might not be the right choice.

I hope this was at least an interesting exploration into one pattern for terse
handling of similar errors and for stacking `io.Writer` interfaces on top of
each other.

Give your dog a pat for me.
