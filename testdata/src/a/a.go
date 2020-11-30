package p

import (
	"os"
	ao "os"
	"os/signal"
)

var c = make(chan os.Signal)
var d = make(chan os.Signal)

func f() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt) // ok
	_ = <-c
}

func g() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt) // want "misuse of unbuffered os.Signal channel as argument to signal.Notify"
	_ = <-c
}

func h() {
	c := make(chan ao.Signal)
	signal.Notify(c, os.Interrupt) // want "misuse of unbuffered os.Signal channel as argument to signal.Notify"
	_ = <-c
}

func i() {
	signal.Notify(d, os.Interrupt) // want "misuse of unbuffered os.Signal channel as argument to signal.Notify"
	f := signal.Notify
	// BAD: this should be warned
	f(d, os.Interrupt)
}
