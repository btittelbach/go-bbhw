/// Author: Bernhard Tittelbach, btittelbach@github  (c) 2014

package bbhw

import "fmt"

// Uses the memory mapped IO to directly interface with AM335x registers.
// Toggles GPIOs about 800 times faster than SysFS.
type MMappedGPIO struct {
	chipid    int
	gpioid    uint
	activelow bool
}

/// Fast MemoryMapped GPIO Stuff -----------------------------------------

// Instantinate a new and fast GPIO controlled using direct access to AM335x registers.
// Takes GPIO numer (same as in sysfs) and direction bbhw.IN or bbhw.OUT
// Only works on AM335x and address compatible SoCs
//
// See http://kilobaser.com/blog/2014-07-15-beaglebone-black-gpios#1gpiopin regarding the numbering of GPIO pins.
func NewMMapedGPIO(number uint, direction int) (gpio *MMappedGPIO) {
	//Set direction and export GPIO via sysfs
	NewSysfsGPIOOrPanic(number, direction).Close()
	gpio = new(MMappedGPIO)

	gpio.chipid, gpio.gpioid = calcGPIOAddrFromLinuxGPIONum(number)
	return gpio
}

func (gpio *MMappedGPIO) CheckDirection() (direction int, err error) {
	mmapreg := getgpiommap()
	input_enabled := mmapreg.memgpiochipreg[gpio.chipid][intgpio_output_enabled_+(gpio.gpioid/8)]&(1<<(gpio.gpioid%8)) > 0
	if input_enabled {
		return IN, nil
	} else {
		return OUT, nil
	}
}

func (gpio *MMappedGPIO) SetDebounce(enable_debounce bool) error {
	mmapreg := getgpiommap()
	if dir, err := gpio.CheckDirection(); dir != IN || err != nil {
		return fmt.Errorf("GPIO %+v is not configured as Input, setting debounce won't have an effect", gpio)
	}
	mmapreg.reglock.Lock()
	debounce_state := mmapreg.memgpiochipreg[gpio.chipid][intgpio_debounceenable_+(gpio.gpioid/8)]
	if enable_debounce {
		debounce_state |= byte(gpio.gpioid % 8)
	} else {
		debounce_state &= byte(^(gpio.gpioid % 8))
	}
	mmapreg.memgpiochipreg[gpio.chipid][intgpio_debounceenable_+(gpio.gpioid/8)] = debounce_state
	mmapreg.reglock.Unlock()
	return nil
}

// This should be about 800 times faster than SysFS GPIOs SetState
// However, even though you can toggle a pin via SysFS,
// setting its state via the memory registers might stop working, once
// a DeviceTreeOverlay for that pin has been loaded (even after you have removed the Overlay)
// in this case: reboot
func (gpio *MMappedGPIO) SetState(state bool) error {
	mmapreg := getgpiommap()
	if state != gpio.activelow {
		mmapreg.memgpiochipreg[gpio.chipid][intgpio_setdataout_+(gpio.gpioid/8)] = 1 << (gpio.gpioid % 8)
	} else {
		mmapreg.memgpiochipreg[gpio.chipid][intgpio_cleardataout_+(gpio.gpioid/8)] = 1 << (gpio.gpioid % 8)
	}

	//sync / flush memory
	// _, _, errno := syscall.Syscall(syscall.SYS_MSYNC, *(*uintptr)(unsafe.Pointer(&mmapreg.memgpiochipreg[gpio.chipid])), uintptr(len(mmapreg.memgpiochipreg[gpio.chipid])), syscall.MS_SYNC)
	// if errno != 0 {
	// 	return syscall.Errno(errno)
	// }
	return nil
}

func (gpio *MMappedGPIO) SetStateNow(state bool) error { return gpio.SetState(state) }

//this inverts the meaning of 0 and 1
//just like in SysFS, this has an immediate effect on the physical output
func (gpio *MMappedGPIO) SetActiveLow(activelow bool) error {
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

// returns true if pin is HIGH and false if pin is LOW i.e. HIGH/LOW signal on input pin
// note that SetActiveLow inverts return value
// internal note: in contrast to SysFS we need to query two different registers depending on the pin direction
func (gpio *MMappedGPIO) GetState() (state bool, err error) {
	mmapreg := getgpiommap()
	var register uint
	if mmapreg.memgpiochipreg[gpio.chipid][intgpio_output_enabled_+(gpio.gpioid/8)]&(1<<(gpio.gpioid%8)) > 0 {
		register = intgpio_datain_ // if DIRECTION==IN
	} else {
		register = intgpio_dataout_ // if DIRECTION==OUT
	}
	state = gpio.activelow != (mmapreg.memgpiochipreg[gpio.chipid][register+(gpio.gpioid/8)]&(1<<(gpio.gpioid%8)) > 0)
	return
}

// not really necessary, but nice to keep same interface as SysfsGPIO
func (gpio *MMappedGPIO) Close() {
	gpio = nil
}
