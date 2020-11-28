package p

import (
	"os"
	ao "os"
	"os/signal"
)

func f() {
	c := make(chan os.Signal, 1) // ok
	signal.Notify(c, os.Interrupt)
	_ = <-c
}

func g() {
	c := make(chan os.Signal) // want "unbuffer os.Signal channel"
	signal.Notify(c, os.Interrupt)
	_ = <-c
}

func h() {
	c := make(chan ao.Signal) // want "unbuffer os.Signal channel"
	signal.Notify(c, os.Interrupt)
	_ = <-c
}
