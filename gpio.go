/// Author: Bernhard Tittelbach, btittelbach@github  (c) 2014

package bbhw

import (
	"log"
	"time"
)

type GPIOControllablePin interface {
	SetState(bool) error
	SetStateNow(bool) error
	GetState() (bool, error)
	CheckDirection() (int, error)
	SetActiveLow(bool) error
}

const (
	IN = iota
	OUT
)

/// GPIOControllablePin Interface and Methods -----------------

func GetStateOrPanic(gpio GPIOControllablePin) bool {
	r, err := gpio.GetState()
	if err != nil {
		panic(err)
	}
	return r
}

func CheckDirectionOrPanic(gpio GPIOControllablePin) int {
	r, err := gpio.CheckDirection()
	if err != nil {
		panic(err)
	}
	return r
}

func Step(gpio GPIOControllablePin, steps uint32, delay time.Duration, abortcheck func() bool) (c uint32, err error) {
	var curstate, oldstate bool
	oldstate, err = gpio.GetState()
	if err != nil {
		log.Println("Step GetState Error:", err)
		return
	}
	//on abort or return set old state
	defer gpio.SetState(oldstate)
	// fix return value
	// due to defer above, we always finish last step, so the number of actual steps takes is roundup(c/2)
	// which is what we want to return
	defer func() { c += c % 2; c /= 2 }()
	curstate = oldstate
	for c = 0; c < steps*2; c++ {
		curstate = !curstate
		err = gpio.SetState(curstate)
		if err != nil {
			log.Println("Step SetState Error:", err)
			return
		}
		time.Sleep(delay)
		if c%2 == 0 && abortcheck != nil && abortcheck() {
			break
		}
	}
	return
}
