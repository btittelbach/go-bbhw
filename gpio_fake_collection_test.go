/// Author: Bernhard Tittelbach, btittelbach@github  (c) 2015

package bbhw

import "testing"

func Test_FakeGPIOCollection(t *testing.T) {
	var gf GPIOCollectionFactory = NewFakeGPIOCollectionFactory()

	var f1 GPIOControllablePinInCollection = gf.NewGPIO(1, OUT)
	var f2 GPIOControllablePinInCollection = gf.NewGPIO(1, IN)

	f1.(*FakeGPIOInCollection).FakeGPIO.ConnectTo(&f2.(*FakeGPIOInCollection).FakeGPIO)
	f1.SetState(false)
	if GetStateOrPanic(f1) != false {
		t.Error("f1 SetState did not work")
	}
	if GetStateOrPanic(f2) != false {
		t.Error("Fake connection to f2 did not work")
	}
	gf.BeginTransactionRecordSetStates()
	f1.SetState(true)
	if GetStateOrPanic(f1) != false || GetStateOrPanic(f2) != false {
		t.Error("BeginTransactionRecordSetStates did not work")
	}
	gf.EndTransactionApplySetStates()
	if GetStateOrPanic(f1) != true || GetStateOrPanic(f2) != true {
		t.Error("BeginTransactionRecordSetStates did not work")
	}
}
