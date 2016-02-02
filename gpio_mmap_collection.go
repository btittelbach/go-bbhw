/// Author: Bernhard Tittelbach, btittelbach@github  (c) 2015

package bbhw

import "sync"

// Uses the memory mapped IO to directly interface with AM335x registers.
// Same as MMappedGPIO, but part of a collection of GPIOs you can set all at once using database-like transactions.
type MMappedGPIOInCollection struct {
	MMappedGPIO
	collection *MMappedGPIOCollectionFactory
}

// Collection of GPIOs. Records SetState() calls after BeginTransactionRecordSetStates() has been called and delays their effect until EndTransactionApplySetStates() is called.
// Use it to toggle many GPIOs in the very same instant.
type MMappedGPIOCollectionFactory struct {
	//4 32bit arrays to be copied to register
	gpios_to_set   []uint32
	gpios_to_clear []uint32
	record_changes bool
	lock           sync.Mutex
}

/// ---------- MMappedGPIOCollectionFactory ---------------

// Create a collection of GPIOs.
// Doubles as factory for the MMappedGPIOInCollection type.
func NewMMapedGPIOCollectionFactory() (gpiocf *MMappedGPIOCollectionFactory) {
	mmapreg := getgpiommap()
	gpiocf = new(MMappedGPIOCollectionFactory)
	gpiocf.gpios_to_set = make([]uint32, len(mmapreg.memgpiochipreg32))
	gpiocf.gpios_to_clear = make([]uint32, len(mmapreg.memgpiochipreg32))
	return gpiocf
}

// Quickly and concurrently (i.e. faster than GPIOChips can react) applies new state
// by writing a 32 bit value to the corresponding CLEARDATA and SETDATA registers for each
// of the four gpio chips.
//
// Note that at LEAST one gpio for each gpiochip has to be exported in sysfs
// in order to active the corresponding gpiochip
// otherwise clocksource of gpiochip remains gated and we hang on receiving SIGBUS immediately after trying to write that register
// see: https://groups.google.com/forum/#!msg/beagleboard/OYFp4EXawiI/Mq6s3sg14HoJ
func (gpiocf *MMappedGPIOCollectionFactory) EndTransactionApplySetStates() {
	mmapreg := getgpiommap()
	gpiocf.lock.Lock()
	defer gpiocf.lock.Unlock()
	for i, _ := range gpiocf.gpios_to_clear {
		if gpiocf.gpios_to_set[i] > 0 || gpiocf.gpios_to_clear[i] > 0 {
			// only set registers which are known to be enabled (i.e. have been set by our code thus have had NewMMapedGPIO called, thus have been exported in sysfs and thus are provided with a clk by the CPU/Linux)
			mmapreg.memgpiochipreg32[i][intgpio_cleardataout_o32_] = gpiocf.gpios_to_clear[i]
			mmapreg.memgpiochipreg32[i][intgpio_setdataout_o32_] = gpiocf.gpios_to_set[i]
		}
		gpiocf.gpios_to_set[i] = 0
		gpiocf.gpios_to_clear[i] = 0
	}
	gpiocf.record_changes = false
}

// Begin recording calls to SetState for later
func (gpiocf *MMappedGPIOCollectionFactory) BeginTransactionRecordSetStates() {
	gpiocf.lock.Lock()
	defer gpiocf.lock.Unlock()
	gpiocf.record_changes = true
}

// Same as NewMMapedGPIO but part of a MMappedGPIOCollectionFactory
func (gpiocf *MMappedGPIOCollectionFactory) NewMMapedGPIO(number uint, direction int) (gpio *MMappedGPIOInCollection) {
	NewSysfsGPIOOrPanic(number, direction).Close()
	gpio = new(MMappedGPIOInCollection)
	gpio.chipid, gpio.gpioid = calcGPIOAddrFromLinuxGPIONum(number)
	gpio.collection = gpiocf
	return gpio
}

func (gpiocf *MMappedGPIOCollectionFactory) NewGPIO(number uint, direction int) GPIOControllablePinInCollection {
	return gpiocf.NewMMapedGPIO(number, direction)
}

/// ------------- MMappedGPIOInCollection Methods -------------------

func (gpio *MMappedGPIOInCollection) SetStateNow(state bool) error {
	return gpio.MMappedGPIO.SetState(state)
}

func (gpio *MMappedGPIOInCollection) SetFutureState(state bool) error {
	gpio.collection.lock.Lock()
	defer gpio.collection.lock.Unlock()
	if gpio.activelow != state {
		gpio.collection.gpios_to_set[gpio.chipid] |= uint32(1 << gpio.gpioid)
		gpio.collection.gpios_to_clear[gpio.chipid] &= ^uint32(1 << gpio.gpioid)
	} else {
		gpio.collection.gpios_to_clear[gpio.chipid] |= uint32(1 << gpio.gpioid)
		gpio.collection.gpios_to_set[gpio.chipid] &= ^uint32(1 << gpio.gpioid)
	}
	return nil
}

/// Checks if State was Set during a transaction but not yet applied
/// state_known returns true if state was set (i.e. either a corresponding bit is set in either clear- or set-register)
/// state returns the future state (i.e. which bit was set)
/// SetActiveLow inverts state but obviously not state_known
/// err returns nil
func (gpio *MMappedGPIOInCollection) GetFutureState() (state_known, state bool, err error) {
	gpio.collection.lock.Lock()
	defer gpio.collection.lock.Unlock()
	state = gpio.collection.gpios_to_set[gpio.chipid]&uint32(1<<gpio.gpioid) > 0
	state_known = state
	if !state_known {
		state_known = gpio.collection.gpios_to_clear[gpio.chipid]&uint32(1<<gpio.gpioid) > 0
	}
	state = state != gpio.activelow
	return
}

func (gpio *MMappedGPIOInCollection) SetState(state bool) error {
	if gpio.collection.record_changes {
		return gpio.SetFutureState(state)
	} else {
		return gpio.SetStateNow(state)
	}
}

//this inverts the meaning of 0 and 1
//just like in SysFS, this has an immediate effect on the physical output
//unless BeginTransactionRecordSetStates() was called beforehand in which case its effect is delayed until EndTransactionApplySetStates()
func (gpio *MMappedGPIOInCollection) SetActiveLow(activelow bool) (err error) {
	if gpio == nil {
		panic("gpio == nil")
	}
	state_known := false
	prev_state := false
	if gpio.collection.record_changes {
		state_known, prev_state, err = gpio.GetFutureState()
		if err != nil {
			return
		}
	}
	if state_known == false {
		prev_state, err = gpio.GetState()
		if err != nil {
			return
		}
	}
	gpio.activelow = activelow
	return gpio.SetState(prev_state)
}
