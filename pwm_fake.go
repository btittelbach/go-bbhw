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

func (pwm *FakePWMPin) Close() {
	pwm = nil
}
