package bbhw

// SysFS managed ADCs ------------------------------------

type FakeADC struct {
	Number uint
	value  uint16
	err    error
}

// Instantinate a new Fake ADC for Simulation
func NewFakeADC(number uint) (adc *FakeADC, err error) {
	adc = new(FakeADC)
	adc.Number = number
	return adc, nil
}

func NewFakeADCOrPanic(number uint) (adc *FakeADC) {
	adc, _ = NewFakeADC(number)
	return
}

func (adc *FakeADC) ReadValue() (value uint16) {
	if adc == nil {
		panic("adc == nil")
	}
	return adc.value
}

func (adc *FakeADC) CheckErrorOccurred() error {
	if adc == nil {
		panic("adc == nil")
	}
	return adc.err
}

func (adc *FakeADC) ReadValueCheckError() (value uint16, err error) {
	value = adc.ReadValue()
	err = adc.CheckErrorOccurred()
	return
}

func (adc *FakeADC) SimulateValue(value uint16, err error) {
	adc.value = value
	adc.err = err
}
