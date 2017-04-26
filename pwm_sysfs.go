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

type BBPWMPin struct {
	fd_period   *os.File
	fd_duty     *os.File
	fd_polarity *os.File
}

type pwmchip struct {
	chip, pwm int
}

var pin_to_pwmchip_map_ = map[string]pwmchip{
	"P9_14": pwmchip{2, 0}, //EHRPWM1A
	"P9_16": pwmchip{2, 1}, //EHRPWM1B
	"P9_21": pwmchip{0, 1}, //EHRPWM0B
	"P9_22": pwmchip{0, 0}, //EHRPWM0A
	"P9_28": pwmchip{7, 0}, //ECAPPWM2
	"P9_29": pwmchip{0, 1}, //EHRPWM0B //same as P9_21
	"P9_31": pwmchip{0, 0}, //EHRPWM0A //same as P9_22
	"P9_42": pwmchip{6, 0}, //ECAPPWM0
	"P8_13": pwmchip{4, 1}, //EHRPWM2B
	"P8_19": pwmchip{4, 0}, //EHRPWM2A
	"P8_34": pwmchip{2, 1}, //EHRPWM1B //same as P9_16
	"P8_36": pwmchip{2, 0}, //EHRPWM1A //same as P9_14
	"P8_45": pwmchip{4, 0}, //EHRPWM2A // same as P8_19
	"P8_46": pwmchip{4, 1}, //EHRPWM2B // same as P8_13

}

func LoadOverlayForSysfsPWM() error {
	err := AddDeviceTreeOverlayIfNotAlreadyLoaded("am33xx_pwm")
	if err == ERROR_DTO_ALREADY_LOADED {
		return nil
	} else {
		return err
	}
}

func findPWMTestDir(bbb_pin string) (tdir string, err error) {
	var ocp_dir string
	if ocp_dir, err = findOCPDir(); err != nil {
		return
	}
	re1 := regexp.MustCompile(filepath.Join(ocp_dir, "(?:bs_)?pwm_test_"+bbb_pin+`(?:\.\d+)?`+"$"))
	err = filepath.Walk(ocp_dir, makeFindDirHelperFunc(&tdir, re1, 5))
	if err == foundit_error_ {
		err = nil
	} else if err == nil {
		err = fmt.Errorf("PWM Directory for %s Not Found", bbb_pin)
	}
	return
}

func findPWMChipDir(chipid int) (string, error) {
	chipdir := fmt.Sprintf("/sys/class/pwm/pwmchip%d/", chipid)
	if fst, err := os.Stat(chipdir); err == nil && fst != nil && fst.IsDir() {
		return chipdir, nil
	} else {
		return "", fmt.Errorf("Directory for PWMChip %d Not Found", chipid)
	}
}

func NewPWMChipPWM(chipid, pwmid int) (pwm *BBPWMPin, err error) {
	var pwmchip_path string
	pwmchip_path, err = findPWMChipDir(chipid)
	if err != nil {
		return
	}
	var exportfile *os.File
	exportfile, err = os.OpenFile(filepath.Join(pwmchip_path, "export"), os.O_WRONLY|os.O_SYNC, 0666)
	defer exportfile.Close()
	var numwritten int
	numwritten, err = exportfile.WriteString(fmt.Sprintf("%d\n", pwmid))
	if err != nil {
		return
	}
	if numwritten < 2 {
		return nil, fmt.Errorf("Could not export pwm %d,%d", chipid, pwmid)
	}
	pwm_path := filepath.Join(pwmchip_path, fmt.Sprintf("pwm%d", pwmid))
	pwm = new(BBPWMPin)
	pwm.fd_period, err = os.OpenFile(filepath.Join(pwm_path, "/period"), os.O_RDWR|os.O_SYNC, 0666)
	if err != nil {
		return
	}
	pwm.fd_duty, err = os.OpenFile(filepath.Join(pwm_path, "/duty_cycle"), os.O_RDWR|os.O_SYNC, 0666)
	if err != nil {
		pwm.fd_period.Close()
		return
	}
	pwm.fd_polarity, err = os.OpenFile(filepath.Join(pwm_path, "/polarity"), os.O_RDWR|os.O_SYNC, 0666)
	if err != nil {
		pwm.fd_period.Close()
		pwm.fd_duty.Close()
		return
	}
	pwm.SetPolarity(false)
	return
}

// Example: StepperPWM, err = NewBBBPWM("P9_16")
func NewBBBPWM(bbb_pin string) (pwm *BBPWMPin, err error) {
	var pwm_path string
	pwm_path, err = findPWMTestDir(bbb_pin)
	if err != nil {
		if pwmchip, lookup_ok := pin_to_pwmchip_map_[bbb_pin]; lookup_ok {
			return NewPWMChipPWM(pwmchip.chip, pwmchip.pwm)
		}
	}
	pwm = new(BBPWMPin)
	var pwm_enable *os.File
	pwm_enable, err = os.OpenFile(pwm_path+"/enable", os.O_RDWR|os.O_SYNC, 0666)
	if err != nil {
		return
	}
	defer pwm_enable.Close()
	pwm_enable.WriteString("1\n")
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
	var val byte = '0'
	if p {
		val = '1'
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

func (pwm *BBPWMPin) Close() {
	pwm.fd_duty.Close()
	pwm.fd_period.Close()
	pwm.fd_polarity.Close()
	pwm = nil
}

func (pwm *BBPWMPin) SetPWMFreq(freq_hz float64) {
	SetPWMFreq(pwm, freq_hz)
}

func (pwm *BBPWMPin) SetPWMFreqDuty(freq_hz, fraction float64) {
	SetPWMFreqDuty(pwm, freq_hz, fraction)
}

func (pwm *BBPWMPin) GetPWMFreqDuty() (freq_hz, fraction float64) {
	return GetPWMFreqDuty(pwm)
}

func (pwm *BBPWMPin) SetStepperRPM(rpm, stepsperrot float64) {
	SetStepperRPM(pwm, rpm*stepsperrot/60.0, 0.1)
}

func (pwm *BBPWMPin) GetStepperRPM(stepsperrot float64) float64 {
	return GetStepperRPM(pwm, stepsperrot)
}
