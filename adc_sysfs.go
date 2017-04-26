package bbhw

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

// SysFS managed ADCs ------------------------------------

type SysfsADC struct {
	Number uint
	fd     *os.File
	err    error
}

func LoadOverlayForSysfsADC() error {
	err := AddDeviceTreeOverlayIfNotAlreadyLoaded("BB-ADC")
	if err == ERROR_DTO_ALREADY_LOADED {
		return nil
	} else {
		return err
	}
}

// Instantinate a new ADC to read through sysfs. Takes ADC AIN numer (same as in sysfs)
func NewSysfsADC(number uint) (adc *SysfsADC, err error) {
	adc = new(SysfsADC)
	adc.Number = number
	ain := fmt.Sprintf("in_voltage%d_raw", number)

	var adc_dir string
	adc_dir, err = findTSCADCDir()
	if err != nil {
		return
	}

	//check if file really exists and open
	adc.fd, err = os.OpenFile(filepath.Join(adc_dir, ain), os.O_RDONLY|os.O_SYNC, 0666)
	if err != nil {
		return nil, err
	}
	return adc, nil
}

// Wrapper around NewSysfsGPIO. Does not return an error but panics instead. Useful to avoid multiple return values.
// This is the function with the same signature as all the other New*GPIO*s
func NewSysfsADCOrPanic(number uint) (adc *SysfsADC) {
	adc, err := NewSysfsADC(number)
	if err != nil {
		panic(err)
	}
	return adc
}

//returns raw SysFs Value.
// In case of BeagleBoneBlack that means actual measured voltage in mV
func (adc *SysfsADC) ReadValue() (value uint16) {
	if adc == nil {
		panic("adc == nil")
	}
	if adc.fd == nil {
		panic("adc.fd == nil")
	}
	_, adc.err = adc.fd.Seek(0, 0)
	if adc.err != nil {
		return
	}

	var numread int
	buf := make([]byte, 16, 16)
	numread, adc.err = adc.fd.Read(buf)
	if adc.err != nil {
		return
	}
	var value64 uint64
	value64, adc.err = strconv.ParseUint(string(buf[0:numread-1]), 10, 16)

	return uint16(value64)
}

func (adc *SysfsADC) CheckErrorOccurred() error {
	if adc == nil {
		panic("adc == nil")
	}
	return adc.err
}

func (adc *SysfsADC) ReadValueCheckError() (value uint16, err error) {
	value = adc.ReadValue()
	err = adc.CheckErrorOccurred()
	return
}

func findTSCADCDir() (adcdir string, err error) {
	var ocp_dir string
	if ocp_dir, err = findOCPDir(); err != nil {
		return
	}
	adcdir = filepath.Join(ocp_dir, "44e0d000.tscadc/TI-am335x-adc/iio:device0/")
	return
}

func findPyADCDir(ain string) (tdir string, err error) {
	var ocp_dir string
	if ocp_dir, err = findOCPDir(); err != nil {
		return
	}
	re1 := regexp.MustCompile(filepath.Join(ocp_dir, `.*`+ain+`\.\d+`+"$"))
	err = filepath.Walk(ocp_dir, makeFindDirHelperFunc(&tdir, re1, 5))
	if err == foundit_error_ {
		err = nil
	} else if err == nil {
		err = fmt.Errorf("ADC Directory for %s Not Found", ain)
	}
	return
}
