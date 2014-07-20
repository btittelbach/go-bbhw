
### go Linux Embedded Fast GPIO, PWM and SerialTTY Library

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

I've also written an article about configuring PINS on the BeagleBone Black using Device-tree Overlays.
[Check It Out](http://kilobaser.com/blog/2014-05-30-beagle-bone-black-devicetreeoverlay-generator)

### Use it



**Keywords:** go golang raspberry beaglebone black white GPIO PWM fast mmap memory mapped am33xx am335xx serial tty serial raw rawtty pinmux 0x194 0x190 0x44E07000 cleardataout setdataout
