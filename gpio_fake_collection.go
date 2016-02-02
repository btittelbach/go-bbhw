/// Author: Bernhard Tittelbach, btittelbach@github  (c) 2015

package bbhw

import (
	"fmt"
	"log"
	"sync"
)

// Uses the memory mapped IO to directly interface with AM335x registers.
// Same as FakeGPIO, but part of a collection of GPIOs you can set all at once using database-like transactions.
type FakeGPIOInCollection struct {
	FakeGPIO
	futureEnable bool
	futureState  bool
	collection   *FakeGPIOCollectionFactory
}

// Collection of GPIOs. Records SetState() calls after BeginTransactionRecordSetStates() has been called and delays their effect until EndTransactionApplySetStates() is called.
// Use it to toggle many GPIOs in the very same instant.
type FakeGPIOCollectionFactory struct {
	//4 32bit arrays to be copied to register
	collection     []*FakeGPIOInCollection
	record_changes bool
	lock           sync.Mutex
}

/// ---------- FakeGPIOCollectionFactory ---------------

// Create a collection of GPIOs.
// Doubles as factory for the FakeGPIOInCollection type.
func NewFakeGPIOCollectionFactory() (gpiocf *FakeGPIOCollectionFactory) {
	gpiocf = new(FakeGPIOCollectionFactory)
	gpiocf.collection = make([]*FakeGPIOInCollection, 0)
	return gpiocf
}

// Apply States recorded with BeginTransactionRecordSetStates
func (gpiocf *FakeGPIOCollectionFactory) EndTransactionApplySetStates() {
	gpiocf.lock.Lock()
	defer gpiocf.lock.Unlock()
	for _, gpio := range gpiocf.collection {
		if gpio.futureEnable {
			gpio.SetStateNow(gpio.futureState)
		}
		gpio.futureEnable = false
	}
	gpiocf.record_changes = false
}

// Begin recording calls to SetState for later
func (gpiocf *FakeGPIOCollectionFactory) BeginTransactionRecordSetStates() {
	gpiocf.lock.Lock()
	defer gpiocf.lock.Unlock()
	gpiocf.record_changes = true
}

// slightly more fancy FakeGPIO for debugging.
// takes a name for easy recognition in debugging output and an optional logger (or nil) of your choice,
// thus you could route debug output of different GPIOs to different destinations
func (gpiocf *FakeGPIOCollectionFactory) NewFakeNamedGPIO(name string, direction int, logTarget *log.Logger) (gpio *FakeGPIOInCollection) {
	gpio = &FakeGPIOInCollection{FakeGPIO: FakeGPIO{name: name, dir: direction, value: false, logTarget: logTarget}, collection: gpiocf}
	gpiocf.lock.Lock()
	gpiocf.collection = append(gpiocf.collection, gpio)
	gpiocf.lock.Unlock()
	return gpio
}

// Same as FakeGPIO but part of a FakeGPIOCollectionFactory
// unfortunately, can't rename this function as it would not be readily interchangeable anymore
func (gpiocf *FakeGPIOCollectionFactory) NewFakeGPIO(number uint, direction int) (gpio *FakeGPIOInCollection) {
	return gpiocf.NewFakeNamedGPIO(fmt.Sprintf("FakeGPIOInCollection(%d,%+v)", number, gpiocf), direction, FakeGPIODefaultLogTarget_)
}

func (gpiocf *FakeGPIOCollectionFactory) NewGPIO(number uint, direction int) GPIOControllablePinInCollection {
	return gpiocf.NewFakeGPIO(number, direction)
}

/// ------------- FakeGPIOInCollection Methods -------------------

func (gpio *FakeGPIOInCollection) SetStateNow(state bool) error {
	if gpio == nil {
		panic("gpio == nil")
	}
	return gpio.FakeGPIO.SetState(state)
}

func (gpio *FakeGPIOInCollection) SetFutureState(state bool) error {
	if gpio == nil {
		panic("gpio == nil")
	}
	gpio.futureEnable = true
	gpio.futureState = state
	return nil
}

func (gpio *FakeGPIOInCollection) GetFutureState() (state_known, state bool, err error) {
	if gpio == nil {
		panic("gpio == nil")
	}
	return gpio.futureEnable, gpio.futureState, nil
}

func (gpio *FakeGPIOInCollection) SetState(state bool) error {
	if gpio == nil {
		panic("gpio == nil")
	}
	if gpio.collection.record_changes {
		return gpio.SetFutureState(state)
	} else {
		return gpio.SetStateNow(state)
	}
}

func (gpio *FakeGPIOInCollection) SetActiveLow(activelow bool) (err error) {
	if gpio == nil {
		panic("gpio == nil")
	}
	return gpio.FakeGPIO.SetActiveLow(activelow)
}
