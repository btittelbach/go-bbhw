/// Author: Bernhard Tittelbach, btittelbach@github  (c) 2014

package bbhw

import "fmt"

// MMappedGPIO Constructor:
// - NewMMapedGPIO
// MMappedGPIO Methods:
// - SetState
// - GetState
// - CheckDirection
// - SetDebounce
// - Close
type MMappedGPIO struct {
	chipid int
	gpioid uint
}

/// Fast MemoryMapped GPIO Stuff -----------------------------------------

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
	if state {
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

func (gpio *MMappedGPIO) GetState() (state bool, err error) {
	mmapreg := getgpiommap()
	// sync / flush memory
	// _, _, errno := syscall.Syscall(syscall.SYS_MSYNC, *(*uintptr)(unsafe.Pointer(&mmapreg.memgpiochipreg[gpio.chipid])), uintptr(len(mmapreg.memgpiochipreg[gpio.chipid])), syscall.MS_SYNC)
	// if errno != 0 {
	// 	err = syscall.Errno(errno)
	// } else {
	// 	state = mmapreg.memgpiochipreg[gpio.chipid][intgpio_setdataout_+(gpio.gpioid/8)]&(1<<(gpio.gpioid%8)) > 0
	// }
	state = mmapreg.memgpiochipreg[gpio.chipid][intgpio_datain_+(gpio.gpioid/8)]&(1<<(gpio.gpioid%8)) > 0
	return
}

// not really necessary, but nice to keep same interface as SysfsGPIO
func (gpio *MMappedGPIO) Close() {
	gpio = nil
}
