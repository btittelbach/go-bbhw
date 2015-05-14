/// Author: Bernhard Tittelbach, btittelbach@github  (c) 2014

package bbhw

// FakeGPIO Constructor:
// - NewFakeGPIO
// FakeGPIO Methods:
// - SetState
// - GetState
// - CheckDirection
// - Close
// - SetDirection
type FakeGPIO struct {
	name  string
	dir   int
	value bool
}

// ----------- Fake GPIO for Testing ----------------

func NewFakeGPIO(name string, direction int) (gpio *FakeGPIO) {
	gpio = &FakeGPIO{name: name, dir: direction, value: false}
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
	return gpio.value, nil
}

func (gpio *FakeGPIO) SetState(state bool) error {
	if gpio == nil {
		panic("gpio == nil")
	}
	if gpio.dir == OUT {
		gpio.value = state
	} else {
		panic("tried to set state on IN gpio")
	}
	return nil
}

func (gpio *FakeGPIO) Close() {
	gpio = nil
}
