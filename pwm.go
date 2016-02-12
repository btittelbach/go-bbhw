package bbhw

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"
)

// PWM Pin Interface

type PWMPin interface {
	SetPolarity(p bool)
	SetPWM(time.Duration, time.Duration)
	GetPWM() (time.Duration, time.Duration)
	DisablePWM()
	SetDuty(float64)
	SetPWMFreqDuty(float64, float64)
	GetPWMFreqDuty() (float64, float64)
	Close()
}

/// --- Interface Functions

func SetStepperRPM(pwm PWMPin, rpm, stepsperrot float64) {
	pwm.SetPWMFreqDuty(rpm*stepsperrot/60.0, 0.1)
}

func GetStepperRPM(pwm PWMPin, stepsperrot float64) float64 {
	freq_hz, _ := pwm.GetPWMFreqDuty()
	return freq_hz / stepsperrot * 60.0
}

func SetPWMFreq(pwm PWMPin, freq_hz float64) {
	pwm.SetPWMFreqDuty(freq_hz, 0.5)
}

//PWM lines

type BBPWMPin struct {
	fd_period   *os.File
	fd_duty     *os.File
	fd_polarity *os.File
}

func findPWMDir(bbb_pin string) (tdir string, err error) {
	path_base := "/sys/devices"
	path_re1 := "^" + path_base + "/ocp" + `\.\d+`
	path_re2 := path_re1 + "/(?:bs_)?pwm_test_" + bbb_pin + `(?:\.\d+)?`
	re1 := regexp.MustCompile(path_re1 + "$")
	re2, err := regexp.Compile(path_re2)
	if err != nil {
		return
	}
	err = filepath.Walk(path_base, makeFindDirHelperFunc(&tdir, path_base, re2, re1))
	if err == foundit_error_ {
		err = nil
	} else if err == nil {
		err = fmt.Errorf("PWM Directory for %s Not Found", bbb_pin)
	}
	return
}

// Example: StepperPWM, err = NewBBBPWM("P9_16")
func NewBBBPWM(bbb_pin string) (pwm *BBPWMPin, err error) {
	var pwm_path string
	pwm_path, err = findPWMDir(bbb_pin)
	if err != nil {
		return
	}
	pwm = new(BBPWMPin)
	pwm.fd_period, err = os.OpenFile(pwm_path+"/period", os.O_RDWR|os.O_SYNC, 0666)
	if err != nil {
		return
	}
	pwm.fd_duty, err = os.OpenFile(pwm_path+"/duty", os.O_RDWR|os.O_SYNC, 0666)
	if err != nil {
		pwm.fd_period.Close()
		return
	}
	pwm.fd_polarity, err = os.OpenFile(pwm_path+"/polarity", os.O_RDWR|os.O_SYNC, 0666)
	if err != nil {
		pwm.fd_period.Close()
		pwm.fd_duty.Close()
		return
	}
	pwm.SetPolarity(false)
	return
}

// Wrapper around NewBBBPWM. Does not return an error but panics instead. Useful to avoid multiple return values.
func NewBBBPWMOrPanic(bbb_pin string) *BBPWMPin {

	pwm, err := NewBBBPWM(bbb_pin)
	if err != nil {
		panic(err)
	}
	return pwm
}

func (pwm *BBPWMPin) SetPolarity(p bool) {
	var val byte = 0
	if p {
		val = 1
	}
	pwm.fd_polarity.Truncate(0)
	pwm.fd_polarity.Write([]byte{val, '\n'})
}

func (pwm *BBPWMPin) DisablePWM() {
	pwm.fd_duty.Truncate(0)
	pwm.fd_duty.Write([]byte{'0', '\n'})
	pwm.SetPolarity(false)
}

func (pwm *BBPWMPin) SetPWM(period, duty time.Duration) {
	if duty > period {
		return
	}
	var buffer []byte = make([]byte, 32)
	pwm.fd_period.Seek(0, 0)
	numread, err := pwm.fd_period.Read(buffer)
	oldperiod, err := strconv.ParseInt(string(buffer[0:numread-1]), 10, 64)
	if err != nil {
		pwm.fd_duty.Truncate(0)
		pwm.fd_duty.WriteString("0\n")
	}

	pwm.fd_duty.Truncate(0)
	pwm.fd_period.Truncate(0)

	if period.Nanoseconds() > oldperiod {
		pwm.fd_period.WriteString(fmt.Sprintf("%d\n", period.Nanoseconds()))
		pwm.fd_duty.WriteString(fmt.Sprintf("%d\n", duty.Nanoseconds()))
	} else {
		pwm.fd_duty.WriteString(fmt.Sprintf("%d\n", duty.Nanoseconds()))
		pwm.fd_period.WriteString(fmt.Sprintf("%d\n", period.Nanoseconds()))
	}
}

func (pwm *BBPWMPin) GetPWM() (period, duty time.Duration) {
	var buffer []byte = make([]byte, 32)
	pwm.fd_period.Seek(0, 0)
	numread, err := pwm.fd_period.Read(buffer)
	oldperiod, err := strconv.ParseInt(string(buffer[0:numread-1]), 10, 64)
	if err == nil {
		period = time.Duration(oldperiod) * time.Nanosecond
	}
	pwm.fd_duty.Seek(0, 0)
	numread, err = pwm.fd_duty.Read(buffer)
	oldduty, err := strconv.ParseInt(string(buffer[0:numread-1]), 10, 64)
	if err == nil {
		duty = time.Duration(oldduty) * time.Nanosecond
	}
	return
}

// set PWM duty to fraction between 0.0 and 1.0
func (pwm *BBPWMPin) SetDuty(fraction float64) {
	if fraction > 1.0 {
		fraction = 1.0
	} else if fraction < 0.0 {
		fraction = 0.0
	}
	var buffer []byte = make([]byte, 32)
	pwm.fd_period.Seek(0, 0)
	numread, err := pwm.fd_period.Read(buffer)
	period, err := strconv.ParseInt(string(buffer[0:numread-1]), 10, 64)
	if err != nil {
		panic(err)
	}
	pwm.fd_duty.Truncate(0)
	pwm.fd_duty.WriteString(fmt.Sprintf("%d\n", uint(float64(period)*fraction)))
}

func (pwm *BBPWMPin) SetPWMFreq(freq_hz float64) {
	SetPWMFreq(pwm, freq_hz)
}

func (pwm *BBPWMPin) SetPWMFreqDuty(freq_hz, fraction float64) {
	if fraction > 1.0 {
		fraction = 1.0
	} else if fraction < 0.0 {
		fraction = 0.0
	}
	period := float64(time.Second) / freq_hz
	pwm.SetPWM(time.Duration(period), time.Duration(period*fraction))
}

func (pwm *BBPWMPin) GetPWMFreqDuty() (freq_hz, fraction float64) {
	period, duty := pwm.GetPWM()
	freq_hz = float64(time.Second) / float64(period)
	fraction = float64(duty) / float64(period)
	return
}

func (pwm *BBPWMPin) Close() {
	pwm.fd_duty.Close()
	pwm.fd_period.Close()
	pwm.fd_polarity.Close()
	pwm = nil

}

func (pwm *BBPWMPin) SetStepperRPM(rpm, stepsperrot float64) {
	SetStepperRPM(pwm, rpm*stepsperrot/60.0, 0.1)
}

func (pwm *BBPWMPin) GetStepperRPM(stepsperrot float64) float64 {
	return GetStepperRPM(pwm, stepsperrot)
}
