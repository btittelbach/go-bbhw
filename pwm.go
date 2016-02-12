package bbhw

import "time"

// PWM Pin Interface

type PWMPin interface {
	SetPolarity(p bool)
	SetPWM(time.Duration, time.Duration)
	GetPWM() (time.Duration, time.Duration)
	DisablePWM()
	Close()
}

/// --- Interface Functions

func SetStepperRPM(pwm PWMPin, rpm, stepsperrot float64) {
	SetPWMFreqDuty(pwm, rpm*stepsperrot/60.0, 0.1)
}

func GetStepperRPM(pwm PWMPin, stepsperrot float64) float64 {
	freq_hz, _ := GetPWMFreqDuty(pwm)
	return freq_hz / stepsperrot * 60.0
}

func SetPWMFreq(pwm PWMPin, freq_hz float64) {
	SetPWMFreqDuty(pwm, freq_hz, 0.5)
}

func SetPWMFreqDuty(pwm PWMPin, freq_hz, fraction float64) {
	if fraction > 1.0 {
		fraction = 1.0
	} else if fraction < 0.0 {
		fraction = 0.0
	}
	period := float64(time.Second) / freq_hz
	pwm.SetPWM(time.Duration(period), time.Duration(period*fraction))
}

func GetPWMFreqDuty(pwm PWMPin) (freq_hz, fraction float64) {
	period, duty := pwm.GetPWM()
	freq_hz = float64(time.Second) / float64(period)
	fraction = float64(duty) / float64(period)
	return
}

// set PWM duty to fraction between 0.0 and 1.0
func SetDuty(pwm PWMPin, fraction float64) {
	if fraction > 1.0 {
		fraction = 1.0
	} else if fraction < 0.0 {
		fraction = 0.0
	}
	period, _ := pwm.GetPWM()
	pwm.SetPWM(time.Duration(period), time.Duration(float64(period.Nanoseconds())*fraction)*time.Nanosecond)
}
