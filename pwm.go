package bbhw

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"
)

//PWM lines

type PWMLine struct {
	fd_period   *os.File
	fd_duty     *os.File
	fd_polarity *os.File
}

func findPWMDir(bbb_pin string) (tdir string, err error) {
	foundit := fmt.Errorf("Success")
	path_base := "/sys/devices"
	path_re1 := "^" + path_base + "/ocp" + `\.\d+`
	path_re2 := path_re1 + "/pwm_test_" + bbb_pin
	re1 := regexp.MustCompile(path_re1 + "$")
	re2, err := regexp.Compile(path_re2)
	if err != nil {
		return
	}
	findPWMDirBBB := func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			return nil
		}
		if path == path_base {
			return nil
		}
		if re2.MatchString(path) {
			tdir = path
			return foundit //foundit
		}
		if !re1.MatchString(path) {
			return filepath.SkipDir //skipdir if not like path_re1
		}
		return nil //continue walking
	}
	err = filepath.Walk(path_base, findPWMDirBBB)
	if err == foundit {
		err = nil
	}
	return
}

// Example: StepperPWM, err = NewBBBPWM("P9_16")
func NewBBBPWM(bbb_pin string) (pwm *PWMLine, err error) {
	var pwm_path string
	pwm_path, err = findPWMDir(bbb_pin)
	if err != nil {
		return
	}
	pwm = new(PWMLine)
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

func (pwm *PWMLine) SetPolarity(p bool) {
	var val byte = 0
	if p {
		val = 1
	}
	pwm.fd_polarity.Truncate(0)
	pwm.fd_polarity.Write([]byte{val, '\n'})
}

func (pwm *PWMLine) DisablePWM() {
	pwm.fd_duty.Truncate(0)
	pwm.fd_duty.Write([]byte{'0', '\n'})
	pwm.SetPolarity(false)
}

func (pwm *PWMLine) SetPWM(period, duty time.Duration) {
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

// set PWM duty to fraction between 0.0 and 1.0
func (pwm *PWMLine) SetDuty(fraction float64) {
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

func (pwm *PWMLine) SetPWMFreq(freq_hz float64) {
	period := time.Duration(float64(time.Second) / freq_hz)
	pwm.SetPWM(period, period/2)
}

func (pwm *PWMLine) SetStepperRPM(rpm, stepsperrot uint32) {
	pwm.SetPWMFreq(float64(rpm*stepsperrot) / 60.0)
}

func (pwm *PWMLine) Close() {
	pwm.fd_duty.Close()
	pwm.fd_period.Close()
	pwm.fd_polarity.Close()
	pwm = nil

}
