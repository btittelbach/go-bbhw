/// Author: Bernhard Tittelbach, btittelbach@github  (c) 2015

package bbhw

import "testing"

func Test_MethodOverriding(t *testing.T) {
	gf := NewMMapedGPIOCollectionFactory()
	g := gf.NewMMapedGPIO(67, OUT)
	g.SetStateNow(false)
	if GetStateOrPanic(g) != false {
		t.Fatal("real state is not false")
	}
	g.SetState(true)
	if GetStateOrPanic(g) != true {
		t.Fatal("real state is not true")
	}
	gf.BeginTransactionRecordSetStates()
	g.SetState(false)
	if GetStateOrPanic(g) == false {
		t.Fatal("real state is false. should not be false before transaction applied")
	}
	t.Logf("gpios_to_clear: %x", gf.gpios_to_clear)
	t.Logf("gpios_to_set: %x", gf.gpios_to_set)
	if gf.gpios_to_clear[g.chipid]&uint32(1<<g.gpioid) == 0 {
		t.Fatal("future clearstate bit is not set.")
	}
	if gf.gpios_to_set[g.chipid]&uint32(1<<g.gpioid) == 1 {
		t.Fatal("future setstate bit is wrongly set.")
	}
	g.SetState(true)
	if gf.gpios_to_clear[g.chipid]&uint32(1<<g.gpioid) == 1 {
		t.Fatal("future clearstate bit is wrongly set.")
	}
	if gf.gpios_to_set[g.chipid]&uint32(1<<g.gpioid) == 0 {
		t.Fatal("future setstate bit is not set.")
	}
	g.SetState(false)
	gf.EndTransactionApplySetStates()
	if GetStateOrPanic(g) != false {
		t.Fatal("real state is not false. should be false after applying transaction")
	}
	if gf.gpios_to_clear[g.chipid] > 0 || gf.gpios_to_set[g.chipid] > 0 {
		t.Fatal("future register should be 0 after EndTransaction")
	}
}

func checkSysfsVersusMMapGPIOFromCollection(gpionum uint, t *testing.T) {
	chipid, gpioid := calcGPIOAddrFromLinuxGPIONum(gpionum)
	t.Logf("Testing sysfs:gpio/gpio%d chip:gpio%d[%d]", gpionum, chipid, gpioid)
	fg := NewMMapedGPIO(gpionum, OUT)
	gf := NewMMapedGPIOCollectionFactory()
	sg := gf.NewMMapedGPIO(gpionum, OUT)

	defer sg.Close()
	defer fg.Close()
	//Step(usr3, 20, time.Duration(200)*time.Millisecond, nil)

	// Test Direction
	d1, err1 := fg.CheckDirection()
	if err1 != nil {
		t.Error(err1.Error())
	}
	if d1 != OUT {
		t.Error("fg.CheckDirection != OUT")
	}
	d2, err2 := sg.CheckDirection()
	if err2 != nil {
		t.Error(err2.Error())
	}
	if d2 != OUT {
		t.Error("sg.CheckDirection != OUT")
	}

	//Test Slow
	gf.BeginTransactionRecordSetStates()
	sg.SetFutureState(true)
	gf.EndTransactionApplySetStates()
	if GetStateOrPanic(sg) != true {
		t.Error("0: sg.GetState() != sg.SetState()")
	}
	gf.BeginTransactionRecordSetStates()
	sg.SetState(false)
	gf.EndTransactionApplySetStates()
	if GetStateOrPanic(sg) != false {
		t.Error("1: sg.GetState() != sg.SetState()")
	}

	// Test Fast
	fg.SetState(true)
	if GetStateOrPanic(fg) != true {
		t.Error("0: fg.GetState() != fg.SetState()")
	}
	fg.SetState(false)
	if GetStateOrPanic(fg) != false {
		t.Error("1: fg.GetState() != fg.SetState()")
	}

	// Test SysFS vs MMapped
	fg.SetState(false)
	if GetStateOrPanic(sg) != false {
		t.Error("1: sg.GetState() != fg.SetState()")
	}
	fg.SetState(true)
	if GetStateOrPanic(sg) != true {
		t.Error("2: sg.GetState() != fg.SetState()")
	}
	gf.BeginTransactionRecordSetStates()
	sg.SetState(false)
	gf.EndTransactionApplySetStates()
	if GetStateOrPanic(fg) != false {
		t.Error("3: sg.GetState() != fg.SetState()")
	}
	gf.BeginTransactionRecordSetStates()
	sg.SetState(true)
	gf.EndTransactionApplySetStates()
	if GetStateOrPanic(fg) != true {
		t.Error("4: sg.GetState() != fg.SetState()")
	}
	gf.BeginTransactionRecordSetStates()
	sg.SetState(false)
	gf.EndTransactionApplySetStates()

}

func Test_MMapGpioFromCollectionVersusSysfsGPIO(t *testing.T) {
	// fg := NewMMapedGPIO(67, OUT)
	// sg := NewSysfsGPIOOrPanic(67, OUT)
	checkSysfsVersusMMapGPIOFromCollection(2, t)
	checkSysfsVersusMMapGPIOFromCollection(3, t)
	checkSysfsVersusMMapGPIOFromCollection(4, t)
	checkSysfsVersusMMapGPIOFromCollection(5, t)
	checkSysfsVersusMMapGPIOFromCollection(50, t)
	checkSysfsVersusMMapGPIOFromCollection(51, t)
	checkSysfsVersusMMapGPIOFromCollection(61, t)
	checkSysfsVersusMMapGPIOFromCollection(67, t)
	checkSysfsVersusMMapGPIOFromCollection(80, t)
	checkSysfsVersusMMapGPIOFromCollection(81, t)
	checkSysfsVersusMMapGPIOFromCollection(88, t)
	checkSysfsVersusMMapGPIOFromCollection(117, t)
}
