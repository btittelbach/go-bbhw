/// Author: Bernhard Tittelbach, btittelbach@github  (c) 2014

package bbhw

import (
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

type GPIOControllablePin interface {
	SetState(bool) error
	GetState() (bool, error)
	CheckDirection() (int, error)
}

// SysfsGPIO Constructor:
// - NewSysfsGPIO
// - NewSysfsGPIOOrPanic
// SysfsGPIO Methods:
// - SetState
// - GetState
// - CheckDirection
// - Close
// - SetDirection
// - ReOpen
type SysfsGPIO struct {
	Number uint
	fd     *os.File
}

// MMappedGPIO Constructor:
// - NewMMapedGPIO
// MMappedGPIO Methods:
// - SetState
// - GetState
// - CheckDirection
// - Close
// - SetDebounce
type MMappedGPIO struct {
	chipid int
	gpioid uint
}

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

const (
	IN = iota
	OUT
)

// SysFS managed GPIO ------------------------------------

func NewSysfsGPIO(number uint, direction int) (gpio *SysfsGPIO, err error) {
	gpio = new(SysfsGPIO)
	gpio.Number = number

	if err := gpio.enable_export(); err != nil {
		return nil, err
	}
	err = gpio.SetDirection(direction)
	if err != nil {
		return nil, err
	}
	//check if file really exists and open for OUT
	gpio.fd, err = os.OpenFile(fmt.Sprintf("/sys/class/gpio/gpio%d/value", gpio.Number), os.O_RDWR|os.O_SYNC, 0666)
	if err != nil {
		return nil, err
	}
	return gpio, nil
}

func NewSysfsGPIOOrPanic(number uint, direction int) (gpio *SysfsGPIO) {
	gpio, err := NewSysfsGPIO(number, direction)
	if err != nil {
		panic(err)
	}
	return gpio
}

func (gpio *SysfsGPIO) ReOpen() (err error) {
	if gpio == nil || gpio.fd == nil {
		return fmt.Errorf("gpio is nil")
	}
	prevfd := gpio.fd
	gpio.fd, err = os.OpenFile(gpio.fd.Name(), os.O_RDWR|os.O_SYNC, 0666)
	if err != nil {
		return
	}
	prevfd.Close()
	return nil
}

func (gpio *SysfsGPIO) enable_export() error {
	if gpio == nil {
		panic("gpio == nil")
	}
	_, err := os.Stat(fmt.Sprintf("/sys/class/gpio/gpio%d", gpio.Number))
	if err == nil {
		// already exported
		return nil
	} else if err != nil && !os.IsNotExist(err) {
		// some other error
		return err
	}
	fd, err := os.OpenFile("/sys/class/gpio/export", os.O_WRONLY|os.O_SYNC, 0666)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(fd, "%d\n", gpio.Number)
	return err
}

func (gpio *SysfsGPIO) CheckDirection() (direction int, err error) {
	var df *os.File
	var n int
	err = nil
	direction = -1
	if gpio == nil {
		panic("gpio == nil")
	}
	filename := fmt.Sprintf("/sys/class/gpio/gpio%d/direction", gpio.Number)
	df, err = os.OpenFile(filename, os.O_RDONLY|os.O_SYNC, 0666)
	if err != nil {
		return
	}
	defer df.Close()
	buf := make([]byte, 16)
	df.Seek(0, 0)
	n, err = df.Read(buf) //go knows how long our buf is, right ??
	if err != nil {
		return
	}
	if n == 0 {
		err = errors.New("wtf ?")
		return
	}
	if string(buf)[0:2] == "in" {
		direction = IN
	} else if string(buf)[0:3] == "out" {
		direction = OUT
	} else {
		err = fmt.Errorf("direction '%s' is neither in nor out !!!", buf)
	}
	return
}

func (gpio *SysfsGPIO) SetDirection(direction int) error {
	if gpio == nil {
		panic("gpio == nil")
	}
	df, err := os.OpenFile(fmt.Sprintf("/sys/class/gpio/gpio%d/direction", gpio.Number),
		os.O_WRONLY|os.O_SYNC, 0666)
	if err != nil {
		return err
	}
	defer df.Close()
	if direction == OUT {
		fmt.Fprintln(df, "out")
	} else {
		fmt.Fprintln(df, "in")
	}
	return nil
}

//this inverts the meaning of 0 and 1 in /sys/class/gpio/gpio*/value
func (gpio *SysfsGPIO) SetActiveLow(activelow bool) error {
	if gpio == nil {
		panic("gpio == nil")
	}
	df, err := os.OpenFile(fmt.Sprintf("/sys/class/gpio/gpio%d/active_low", gpio.Number),
		os.O_WRONLY|os.O_SYNC, 0666)
	if err != nil {
		return err
	}
	defer df.Close()
	if activelow {
		fmt.Fprintln(df, "1")
	} else {
		fmt.Fprintln(df, "0")
	}
	return nil
}

func (gpio *SysfsGPIO) GetState() (state bool, err error) {
	if gpio == nil {
		panic("gpio == nil")
	}
	var n int
	if gpio.fd == nil {
		panic("gpio.fd == nil")
	}
	_, err = gpio.fd.Seek(0, 0)

	// if err = gpio.ReOpen(); err != nil {
	// 	return
	// }
	buf := make([]byte, 16)
	n, err = gpio.fd.Read(buf) //go knows how long our buffer is, right ??
	if err != nil {
		return
	}
	if n != 2 {
		err = errors.New("wtf ?")
		return
	}
	if buf[0] == '1' {
		state = true
	} else {
		state = false
	}
	return
}

func (gpio *SysfsGPIO) SetState(state bool) error {
	if gpio == nil || gpio.fd == nil {
		panic("gpio == nil")
	}
	v := "0"
	if state {
		v = "1"
	}
	gpio.fd.Truncate(0)
	_, err := fmt.Fprintln(gpio.fd, v)
	return err
}

func (gpio *SysfsGPIO) Close() {
	gpio.fd.Close()
	gpio = nil
}

/// Fast MemoryMapped GPIO Stuff -----------------------------------------
/// This ONLY works on the BeagleBone or similar AM335xx devices !!!

type mappedRegisters struct {
	memfd          *os.File
	memgpiochipreg [][]byte
	reglock        sync.Mutex
}

var mmaped_gpio_register_ *mappedRegisters
var mmaped_lock_ sync.Mutex

const ( // AM335x Memory Addresses
	gpio0_offset_                = 0x44E07000
	gpio1_offset_                = 0x4804C000
	gpio_pagesize_               = 0x1000 //4KiB
	spinlock_offset_             = 0x480CA000
	spinlock_pagesize_           = 0x1000 //4KiB
	gpio2_offset_                = 0x481AC000
	gpio3_offset_                = 0x481AE000
	pinmux_controlmodule_offset_ = 0x44E10000
	intgpio_setdataout_          = 0x194
	intgpio_cleardataout_        = 0x190
	intgpio_datain_              = 0x138
	intgpio_debounceenable_      = 0x150
	intgpio_debouncetime_        = 0x154
	intgpio_output_enabled_      = 0x134
)

func verifyAddrIsTIOmap4(addr uint) bool {
	filename := fmt.Sprintf("/proc/device-tree/ocp/gpio@%x/compatible", addr)
	pf, err := os.OpenFile(filename, os.O_RDONLY, 0666)
	if err != nil {
		return false
	}
	defer pf.Close()
	buf := make([]byte, 32)
	pf.Seek(0, 0)
	n, err := pf.Read(buf) //go knows how long our buf is, right ??
	if err != nil || n == 0 {
		return false
	}
	return string(buf)[0:13] == "ti,omap4-gpio"
}

func newGPIORegMMap() (mmapreg *mappedRegisters, err error) {
	//Verify our memory addresses are actually correct
	if !(verifyAddrIsTIOmap4(gpio0_offset_) && verifyAddrIsTIOmap4(gpio1_offset_) && verifyAddrIsTIOmap4(gpio2_offset_) && verifyAddrIsTIOmap4(gpio3_offset_)) {
		return nil, fmt.Errorf("Looks like we aren't on a AM33xx CPU! Please check your Datasheet and update the code (github) or stick to the SysFSGPIOs")
	}
	mmapreg = new(mappedRegisters)
	mmapreg.memgpiochipreg = make([][]byte, 4)
	//Now MemoryMap
	mmapreg.memfd, err = os.OpenFile("/dev/mem", os.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}
	mmapreg.memgpiochipreg[0], err = syscall.Mmap(int(mmapreg.memfd.Fd()), gpio0_offset_, gpio_pagesize_, syscall.PROT_WRITE|syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		mmapreg.close()
		return nil, err
	}
	mmapreg.memgpiochipreg[1], err = syscall.Mmap(int(mmapreg.memfd.Fd()), gpio1_offset_, gpio_pagesize_, syscall.PROT_WRITE|syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		mmapreg.close()
		return nil, err
	}
	mmapreg.memgpiochipreg[2], err = syscall.Mmap(int(mmapreg.memfd.Fd()), gpio2_offset_, gpio_pagesize_, syscall.PROT_WRITE|syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		mmapreg.close()
		return nil, err
	}
	mmapreg.memgpiochipreg[3], err = syscall.Mmap(int(mmapreg.memfd.Fd()), gpio3_offset_, gpio_pagesize_, syscall.PROT_WRITE|syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		mmapreg.close()
		return nil, err
	}
	return mmapreg, nil
}

func (mmapreg *mappedRegisters) close() {
	if mmapreg == nil {
		return
	}
	for i := 0; i < len(mmapreg.memgpiochipreg); i++ {
		if mmapreg.memgpiochipreg[i] == nil {
			continue
		}
		//dh := (*reflect.SliceHeader)(unsafe.Pointer(mmapreg.memgpiochipreg[i]))
		_, _, errno := syscall.Syscall(syscall.SYS_MUNMAP, *(*uintptr)(unsafe.Pointer(&mmapreg.memgpiochipreg[i])), unsafe.Sizeof(mmapreg.memgpiochipreg[i]), 0)
		if errno != 0 {
			panic(syscall.Errno(errno))
		}
	}
	mmapreg.memfd.Close()
	mmapreg = nil
}

func (mmapreg *mappedRegisters) setDebounceTime(gpiochip int, dbt byte) error {
	if mmapreg == nil {
		return fmt.Errorf("mmapreg object does not exist")
	}
	if gpiochip < 0 || gpiochip >= len(mmapreg.memgpiochipreg) {
		return fmt.Errorf("gpiochip id %d is out of bounds [0,%d]", gpiochip, len(mmapreg.memgpiochipreg)-1)
	}
	if mmapreg.memgpiochipreg[gpiochip] == nil {
		return fmt.Errorf("memgpiochipreg[%d] == nil", gpiochip)
	}
	mmapreg.memgpiochipreg[gpiochip][intgpio_debouncetime_] = dbt
	return nil
}

func getgpiommap() *mappedRegisters {
	if mmaped_gpio_register_ == nil {
		var err error
		mmaped_gpio_register_, err = newGPIORegMMap()
		if err != nil {
			panic(err)
		}
	}
	return mmaped_gpio_register_
}

//careful with this function! never call it
//if there's a chance some routine might still be using fast gpios
//If in Doubt: Never Call It
func MMappedGPIOCleanup() {
	if mmaped_gpio_register_ != nil {
		mmaped_gpio_register_.close()
	}
}

func calcGPIOAddrFromLinuxGPIONum(number uint) (chipid int, gpioid uint) {
	chipid = int(number) / 32
	gpioid = number % 32
	return
}

//Todo: Try Set PinMux by mmapping pinmux_controlmodule_offset_
// http://rampic.com/beagleboneblack/

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

func Step(gpio GPIOControllablePin, steps uint32, delay time.Duration, abortcheck func() bool) (err error) {
	var curstate, oldstate bool
	oldstate, err = gpio.GetState()
	if err != nil {
		log.Println("Step GetState Error:", err)
		return
	}
	defer gpio.SetState(oldstate)
	curstate = oldstate
	var c uint32
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
