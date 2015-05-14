package bbhw

import (
	"fmt"
	"os"
	"sync"
	"syscall"
	"unsafe"
)

/// This ONLY works on the BeagleBone or similar AM335xx devices !!!

type mappedRegisters struct {
	memfd          *os.File
	memgpiochipreg [][]byte
	reglock        sync.Mutex
}

var mmaped_gpio_register_ *mappedRegisters
var mmaped_lock_ sync.Mutex

const ( // AM335x Memory Addresses
	omap4_gpio0_offset_           = 0x44E07000
	omap4_gpio1_offset_                = 0x4804C000
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
	if !(verifyAddrIsTIOmap4(omap4_gpio0_offset_) && verifyAddrIsTIOmap4(omap4_gpio1_offset_) && verifyAddrIsTIOmap4(gpio2_offset_) && verifyAddrIsTIOmap4(gpio3_offset_)) {
		return nil, fmt.Errorf("Looks like we aren't on a AM33xx CPU! Please check your Datasheet and update the code (github) or stick to the SysFSGPIOs")
	}
	mmapreg = new(mappedRegisters)
	mmapreg.memgpiochipreg = make([][]byte, 4)
	//Now MemoryMap
	mmapreg.memfd, err = os.OpenFile("/dev/mem", os.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}
	mmapreg.memgpiochipreg[0], err = syscall.Mmap(int(mmapreg.memfd.Fd()), omap4_gpio0_offset_, gpio_pagesize_, syscall.PROT_WRITE|syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		mmapreg.close()
		return nil, err
	}
	mmapreg.memgpiochipreg[1], err = syscall.Mmap(int(mmapreg.memfd.Fd()), omap4_gpio1_offset_, gpio_pagesize_, syscall.PROT_WRITE|syscall.PROT_READ, syscall.MAP_SHARED)
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
