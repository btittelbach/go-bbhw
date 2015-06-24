/// Author: Bernhard Tittelbach, btittelbach@github  (c) 2014

package bbhw

import (
	"fmt"
	"log"
)

// FakeGPIO Constructor:
// - NewFakeGPIO
// FakeGPIO Methods:
// - SetState
// - GetState
// - CheckDirection
// - Close
// - SetDirection
// - SetActiveLow
type FakeGPIO struct {
	name        string
	dir         int
	value       bool
	activelow   bool
	logTarget   *log.Logger
	connectedTo []*FakeGPIO
}

type FakeGPIONullWriter struct{}

func (n *FakeGPIONullWriter) Write(p []byte) (int, error) { return len(p), nil }

var FakeGPIODefaultLogTarget_ *log.Logger

func init() {
	FakeGPIODefaultLogTarget_ = log.New(&FakeGPIONullWriter{}, "", 0)
}

// ----------- Fake GPIO for Testing ----------------

func NewFakeGPIO(gpionum uint, direction int) (gpio *FakeGPIO) {
	return NewFakeNamedGPIO(fmt.Sprintf("FakeGPIO(%d)", gpionum), direction, nil)
}

func NewFakeNamedGPIO(name string, direction int, logTarget *log.Logger) (gpio *FakeGPIO) {
	gpio = &FakeGPIO{name: name, dir: direction, value: false, logTarget: logTarget}
	return
}

func (gpio *FakeGPIO) CheckDirection() (direction int, err error) {
	return gpio.dir, nil
}

func (gpio *FakeGPIO) SetDirection(direction int) error {
	if !(direction == IN || direction == OUT) {
		panic("direction neither IN nor OUT")
	}
	gpio.dir = direction
	return nil
}

func (gpio *FakeGPIO) GetState() (state bool, err error) {
	return gpio.activelow != gpio.value, nil
}

func (gpio *FakeGPIO) SetState(state bool) error {
	if gpio == nil {
		panic("gpio == nil")
	}
	if gpio.dir == OUT {
		gpio.value = gpio.activelow != state
		gpio.log("set to virtual electrical state >%+v<", gpio.value)
		if gpio.connectedTo != nil {
			for _, othergpio := range gpio.connectedTo {
				if othergpio == nil {
					continue
				}
				othergpio.FakeInput(gpio.value)
			}
		}
	} else {
		panic("tried to set state on IN gpio")
	}
	return nil
}

//this inverts the meaning of virtual 0 and 1
func (gpio *FakeGPIO) SetActiveLow(activelow bool) error {
	if gpio == nil {
		panic("gpio == nil")
	}
	prev_state, err := gpio.GetState()
	if err != nil {
		return err
	}
	gpio.activelow = activelow
	return gpio.SetState(prev_state)
}

func (gpio *FakeGPIO) SetStateNow(state bool) error { return gpio.SetState(state) }

func (gpio *FakeGPIO) Close() {
	gpio = nil
}

func (gpio *FakeGPIO) ConnectTo(conn ...*FakeGPIO) {
	gpio.connectedTo = conn
	if gpio.connectedTo != nil {
		var gpionames string
		for _, othergpio := range gpio.connectedTo {
			if othergpio == nil {
				continue
			}
			dir := "IN"
			if othergpio.dir == OUT {
				dir = "OUT"
			}
			gpionames += " " + othergpio.name + "(" + dir + ")"
		}
		gpio.log("now connected to" + gpionames)
	}

}

func (gpio *FakeGPIO) FakeInput(state bool) error {
	if gpio == nil {
		panic("gpio == nil")
	}
	if gpio.dir == IN {
		gpio.log("faking input >%+v<", state)
		gpio.value = state
	} else {
		panic("tried to fake input for output gpio")
	}
	return nil
}

func (gpio *FakeGPIO) log(fmt string, attr ...interface{}) {
	logT := gpio.logTarget
	if logT == nil {
		logT = FakeGPIODefaultLogTarget_
	}
	dir := "IN"
	if gpio.dir == OUT {
		dir = "OUT"
	}
	logT.Printf("FakeGPIO %s(%s): "+fmt, append([]interface{}{gpio.name, dir}, attr...)...)

}
