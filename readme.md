go-BeagleBone Hardware
=======

BeagleBone Hardware Controls (GPIO,PWM,TTY,DTO) for the go language
(and for other Linux Embedded Devices)

## About

This is a small library to make dealing with Linux embedded devices in go easier.
It is mainly written for the BeagleBone Black but with the exception of the Memory Mapped Fast GPIO,
is works well on the Raspberry or other embedded Linux devices (e.g.: TP-Link, PC-Engines) as well.

The library currently does three things:

- It implements memory mapped GPIOs for the AM335xx, the beagle bone CPU, which allows us to toggle about 800 times faster than sysfs controlled GPIOs.
- For other Linux embedded devices it implements a comprehensive normal GPIO library
- It provides an extensive interface to the BeagleBone's PWM control
- It provides tools to interface with character oriented rawtty serial devices (as opposed to line oriented)

Before writing it, I had a look at aqua's raspberry lib, which was too basic for my needs,
but which is why the SysFSGPIO Interface looks similar. You should check out his [repository](https://github.com/aqua/raspberrypi)
if you are looking for go-support for OneWire / DS18x20 devices.

I've also written two blogs about [using PINS on the BeagleBone Black](http://kilobaser.com/blog/2014-07-15-beaglebone-black-gpios) and [making Device-Tree Overlays](http://kilobaser.com/blog/2014-07-28-beaglebone-black-devicetreeoverlay-generator) to configure the BeagleBone Black.

## Usage
```
#> go get github.com/btittelbach/go-bbhw
```

```
import "github.com/btittelbach/go-bbhw"

bbhw.
```

View the API docs [here](http://godoc.org/github.com/btittelbach/go-bbhw)



### Using GPIOs
Control GPIOs using the ```GPIOControllablePin``` interface for which **four** implemenations are provided

```golang
type GPIOControllablePin interface {
    SetState(bool) error
    SetStateNow(bool) error
    GetState() (bool, error)
    CheckDirection() (int, error)
    SetActiveLow(bool) error
}

```

#### Fake GPIO
Use FakeGPIO for testing and debugging. Does not actually toogle GPIOs and works even on your normal computer.

```go
func NewFakeGPIO(gpionum uint, direction int) (gpio *FakeGPIO)
    same signature as all the other New*GPIO implementations. logs to
    FakeGPIODefaultLogTarget_ which is an exported field and thus you can
    set it to point to the log.Logger of your choice

func NewFakeNamedGPIO(name string, direction int, logTarget *log.Logger) (gpio *FakeGPIO)
    slightly more fancy FakeGPIO for debugging. takes a name for easy
    recognition in debugging output and an optional logger (or nil) of your
    choice, thus you could route debug output of different GPIOs to
    different destinations

```

#### SysFS GPIO
Uses the ```/sys/class/gpio/**/*``` file-interface provided by the linux kernel.
Slightly slower than mmapped implementations but will work on any linux system with GPIOs.

```go
func NewSysfsGPIO(number uint, direction int) (gpio *SysfsGPIO, err error)
    Instantinate a new GPIO to control through sysfs. Takes GPIO numer (same
    as in sysfs) and direction bbhw.IN or bbhw.OUT

func NewSysfsGPIOOrPanic(number uint, direction int) (gpio *SysfsGPIO)
    Wrapper around NewSysfsGPIO. Does not return an error but panics
    instead. Useful to avoid multiple return values. This is the function
    with the same signature as all the other New*GPIO*s
```

#### MemoryMapped GPIO
Uses the memory mapped IO to directly interface with AM335x registers.
Toggles GPIOs about 800 times faster than SysFS.

```go
func NewMMapedGPIO(number uint, direction int) (gpio *MMappedGPIO)
    Instantinate a new and fast GPIO controlled using direct access to
    AM335x registers. Takes GPIO numer (same as in sysfs) and direction
    bbhw.IN or bbhw.OUT Only works on AM335x and address compatible SoCs
```

####  Collection of MemoryMapped GPIOs
Same as MMappedGPIO, but part of a collection of GPIOs you can set all at once using database-like transactions.
Records SetState() calls after BeginTransactionRecordSetStates() has been called and delays their effect until EndTransactionApplySetStates() is called. Use it to toggle many GPIOs in the very same instant.

```go
func NewMMapedGPIOCollectionFactory() (gpiocf *MMappedGPIOCollectionFactory)
    Create a collection of GPIOs. Doubles as factory for the
    MMappedGPIOInCollection type.

func (gpiocf *MMappedGPIOCollectionFactory) NewMMapedGPIO(number uint, direction int) (gpio *MMappedGPIOInCollection)
    Same as NewMMapedGPIO but part of a MMappedGPIOCollectionFactory
```
## Keywords
go golang raspberry beaglebone black white GPIO PWM fast mmap memory mapped am33xx am335xx serial tty serial raw rawtty pinmux 0x194 0x190 0x44E07000 cleardataout setdataout
