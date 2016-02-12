package bbhw

import "time"

// Fake PWM for Testing

type FakePWMPin struct {
	name     string
	period   time.Duration
	duty     time.Duration
	polarity bool
}

// Example: StepperPWM, err = NewBBBPWM("P9_16")
func NewFakePWM(name string) (pwm *FakePWMPin, err error) {
	pwm = new(FakePWMPin)
	pwm.name = name
	pwm.SetPolarity(false)
	err = nil
	return
}

// Wrapper around NewFakePWM. Does not return an error but panics instead. Useful to avoid multiple return values.
func NewFakePWMOrPanic(bbb_pin string) *FakePWMPin {
	pwm, _ := NewFakePWM(bbb_pin)
	return pwm //no need to check error
}

func (pwm *FakePWMPin) SetPolarity(p bool) {
	pwm.polarity = p
}

func (pwm *FakePWMPin) DisablePWM() {
	pwm.duty = 0
	pwm.polarity = false
}

func (pwm *FakePWMPin) SetPWM(period, duty time.Duration) {
	if duty > period {
		return
	}
	pwm.duty = duty
	pwm.period = period
}

func (pwm *FakePWMPin) GetPWM() (period, duty time.Duration) {
	return pwm.period, pwm.duty
}

// set PWM duty to fraction between 0.0 and 1.0
func (pwm *FakePWMPin) SetDuty(fraction float64) {
	if fraction > 1.0 {
		fraction = 1.0
	} else if fraction < 0.0 {
		fraction = 0.0
	}
	pwm.duty = time.Duration(int64(float64(pwm.period.Nanoseconds())*fraction)) * time.Nanosecond
}

func (pwm *FakePWMPin) SetPWMFreq(freq_hz float64) {
	SetPWMFreq(pwm, freq_hz)
}

func (pwm *FakePWMPin) SetPWMFreqDuty(freq_hz, fraction float64) {
	if fraction > 1.0 {
		fraction = 1.0
	} else if fraction < 0.0 {
		fraction = 0.0
	}
	period := float64(time.Second) / freq_hz
	pwm.SetPWM(time.Duration(period), time.Duration(period*fraction))
}

func (pwm *FakePWMPin) GetPWMFreqDuty() (freq_hz, fraction float64) {
	period, duty := pwm.GetPWM()
	freq_hz = float64(time.Second) / float64(period)
	fraction = float64(duty) / float64(period)
	return
}

func (pwm *FakePWMPin) Close() {
	pwm = nil
}

func (pwm *FakePWMPin) SetStepperRPM(rpm, stepsperrot float64) {
	SetStepperRPM(pwm, rpm*stepsperrot/60.0, 0.1)
}

func (pwm *FakePWMPin) GetStepperRPM(stepsperrot float64) float64 {
	return GetStepperRPM(pwm, stepsperrot)
}
