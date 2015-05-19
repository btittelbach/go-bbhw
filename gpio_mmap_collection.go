/// Author: Bernhard Tittelbach, btittelbach@github  (c) 2015

package bbhw

type MMappedGPIOInCollection struct {
	MMappedGPIO
	collection *MMappedGPIOCollectionFactory
}

type MMappedGPIOCollectionFactory struct {
	//4 * 4byte arrays to be copied
	gpios_to_set   [][]byte
	gpios_to_clear [][]byte
	record_changes bool
}

/// ---------- MMappedGPIOCollectionFactory ---------------

func NewMMapedGPIOCollectionFactory() (gpiocf *MMappedGPIOCollectionFactory) {
	mmapreg := getgpiommap()
	gpiocf = new(MMappedGPIOCollectionFactory)
	gpiocf.gpios_to_set = make([][]byte, len(mmapreg.memgpiochipreg))
	gpiocf.gpios_to_clear = make([][]byte, len(mmapreg.memgpiochipreg))
	for i, _ := range mmapreg.memgpiochipreg {
		gpiocf.gpios_to_set[i] = make([]byte, 4)
		gpiocf.gpios_to_clear[i] = make([]byte, 4)
	}
	return gpiocf
}

func (gpiocf *MMappedGPIOCollectionFactory) EndTransactionApplySetStates() {
	mmapreg := getgpiommap()
	for i, _ := range gpiocf.gpios_to_set {
		//FIXME: do a 32bit write/move instead of 4 bytewise
		for j := 0; j < len(gpiocf.gpios_to_set[i]); j++ {
			mmapreg.memgpiochipreg[i][intgpio_setdataout_+j] = gpiocf.gpios_to_set[i][j]
			mmapreg.memgpiochipreg[i][intgpio_cleardataout_+j] = gpiocf.gpios_to_clear[i][j]
			gpiocf.gpios_to_set[i][j] = 0
			gpiocf.gpios_to_clear[i][j] = 0
		}
	}
	gpiocf.record_changes = false
}

func (gpiocf *MMappedGPIOCollectionFactory) BeginTransactionRecordSetStates() {
	gpiocf.record_changes = true
}

func (gpiocf *MMappedGPIOCollectionFactory) NewMMapedGPIO(number uint, direction int) (gpio *MMappedGPIOInCollection) {
	NewSysfsGPIOOrPanic(number, direction).Close()
	gpio = new(MMappedGPIOInCollection)
	gpio.chipid, gpio.gpioid = calcGPIOAddrFromLinuxGPIONum(number)
	gpio.collection = gpiocf
	return gpio
}

/// ------------- MMappedGPIOInCollection Methods -------------------

func (gpio *MMappedGPIOInCollection) SetStateNow(state bool) error {
	return gpio.MMappedGPIO.SetState(state)
}

func (gpio *MMappedGPIOInCollection) SetFutureState(state bool) error {
	if state {
		gpio.collection.gpios_to_set[gpio.chipid][gpio.gpioid/8] |= byte(1 << (gpio.gpioid % 8))
		gpio.collection.gpios_to_clear[gpio.chipid][gpio.gpioid/8] &= ^byte(1 << (gpio.gpioid % 8))
	} else {
		gpio.collection.gpios_to_clear[gpio.chipid][gpio.gpioid/8] |= byte(1 << (gpio.gpioid % 8))
		gpio.collection.gpios_to_set[gpio.chipid][gpio.gpioid/8] &= ^byte(1 << (gpio.gpioid % 8))
	}
	return nil
}

/// Checks if State was Set during a transaction but not yet applied
/// state_known returns true if state was set
/// state returns the future state
/// err returns nil
func (gpio *MMappedGPIOInCollection) GetFutureState() (state_known, state bool, err error) {
	state = gpio.collection.gpios_to_set[gpio.chipid][(gpio.gpioid/8)]&(1<<(gpio.gpioid%8)) > 0
	state_known = state
	if !state_known {
		state_known = gpio.collection.gpios_to_clear[gpio.chipid][(gpio.gpioid/8)]&(1<<(gpio.gpioid%8)) > 0
	}
	return
}

func (gpio *MMappedGPIOInCollection) SetState(state bool) error {
	if gpio.collection.record_changes {
		return gpio.SetFutureState(state)
	} else {
		return gpio.SetStateNow(state)
	}
}
