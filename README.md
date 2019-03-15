# OnEdge &nbsp; <img src="logo/onedge.png" width="189" height="52" />

OnEdge is a library for detecting certain improper uses of the
[Defer, Panic, and Recover](https://blog.golang.org/defer-panic-and-recover) pattern in Go programs.
OnEdge is lightweight in that it is easy to incorporate into your project and tries to stay out of your
way as much as possible.

## What sort of problems does OnEdge detect?

OnEdge detects global state changes that occur between (1) the entry point to a function that `defer`s a
call to `recover` and (2) the point at which `recover` is called.  Often, such global state changes are
unintentional, e.g., the programmer didn't realize that code executed before a `panic` could have a
lasting effect on their program's behavior.

## How does OnEdge work?

OnEdge reduces the problem of finding such global state changes to one of race detection.  When the
program enters a function that `defer`s a call to `recover`, OnEdge launches a _shadow thread_.  If that
function later `panic`s, then the function is re-executed in the shadow thread.  If doing so causes the
shadow thread to make a global state change before calling `recover`, then that change appears as a data
race and is reported by [Go's race detector](https://golang.org/doc/articles/race_detector.html).

When Go's race detector is disabled, OnEdge does nothing.

## How do you incorporate OnEdge into your project?

To incorporate OnEdge into your project, you must do three things.

1. **Wrap functions that `defer` calls to `recover` in `onedge.WrapFunc(func() {` ... `})`**, e.g.,
```
onedge.WrapFunc(func() { handle(request) })
```

2. **Within wrapped functions, wrap calls to `recover` in `onedge.WrapRecover(` ... `)`**, e.g.,
```
func handle(request Request) {
    defer func() {
        if r := onedge.WrapRecover(recover()); r != nil {
          log.Println(r)
        }
    }()
    ...
    // Panicky code that potentially modifies global state
    ...
}
```

3. **Run your program with Go's race detector
[enabled](https://golang.org/doc/articles/race_detector.html#Usage)**, e.g.,
```
$ go run -race mysrc.go
```

Data races will be reported for global state changes that occur
* after entry to a function wrapped by `WrapFunc`
* but before a `recover` wrapped by `WrapRecover`.

## Limitations

1. OnEdge is a dynamic analysis, and like all dynamic analyses, its effectiveness depends upon the
workload to which you subject your program.  In other words, for OnEdge to detect some global state
change, you must provide inputs to your program that cause it to make that global state change.

2. For some programs, OnEdge's results are non-deterministic, i.e., OnEdge could report a global state
change for some runs of the program, but not for others.  We suspect this is because
[ThreadSanitizer](https://github.com/google/sanitizers) (on which Go's race detector is built) is itself
non-deterministic.  However, further investigation into this issue is needed.

3. If your program is multithreaded, then use of OnEdge can cause spurious data races to be reported.
This has to do with how the main thread communicates with shadow threads.  If you think that your
program may contain a legitimate data race, then we recommend that you deal with that before enabling
OnEdge.

4. While nested uses to `WrapFunc` are supported, they can cause data races to be reported in OnEdge
itself.  This is because OnEdge must, e.g., keep track of shadow threads, and doing so involves
modifying the global state.  In theory, this problem could be solved modifying the Go compiler (e.g.,
[here](https://github.com/golang/go/blob/master/src/cmd/compile/internal/gc/racewalk.go)) to ignore the
OnEdge package.  But modifying the Go compiler seems like a rather heavy handed solution to an
infrequently occurring problem.  So, for now, we recommend that users simply ignore any reported data
races involving OnEdge's code itself.

## References

* Andrew Gerrand. [Defer, Panic, and Recover](https://blog.golang.org/defer-panic-and-recover). The Go Blog, 4 August 2010.

* The Go Authors. [Data Race Detector](https://golang.org/doc/articles/race_detector.html).

* Google. [AddressSanitizer, ThreadSanitizer, MemorySanitizer](https://github.com/google/sanitizers).

* Kavya Joshi. [Looking inside a Race Detector](https://www.infoq.com/presentations/go-race-detector). QCon, 10 March 2017.

* Dmitry Vyukov and Andrew Gerrand. [Introducing the Go Race Detector](https://blog.golang.org/race-detector). The Go Blog, 26 June 2013.
