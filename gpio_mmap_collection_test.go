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
	t.Logf("gpios_to_clear: %x%x%x%x", gf.gpios_to_clear[0], gf.gpios_to_clear[1], gf.gpios_to_clear[2], gf.gpios_to_clear[3])
	t.Logf("gpios_to_set: %x%x%x%x", gf.gpios_to_set[0], gf.gpios_to_set[1], gf.gpios_to_set[2], gf.gpios_to_set[3])
	if gf.gpios_to_clear[g.chipid][g.gpioid/8]&byte(1<<g.gpioid) == 0 {
		t.Fatal("future clearstate bit is not set.")
	}
	if gf.gpios_to_set[g.chipid][g.gpioid/8]&byte(1<<g.gpioid) == 1 {
		t.Fatal("future setstate bit is wrongly set.")
	}
	g.SetState(true)
	if gf.gpios_to_clear[g.chipid][g.gpioid/8]&byte(1<<g.gpioid) == 1 {
		t.Fatal("future clearstate bit is wrongly set.")
	}
	if gf.gpios_to_set[g.chipid][g.gpioid/8]&byte(1<<g.gpioid) == 0 {
		t.Fatal("future setstate bit is not set.")
	}
	g.SetState(false)
	gf.EndTransactionApplySetStates()
	if GetStateOrPanic(g) != false {
		t.Fatal("real state is not false. should be false after applying transaction")
	}
	if gf.gpios_to_clear[g.chipid][g.gpioid/8] > 0 || gf.gpios_to_set[g.chipid][g.gpioid/8] > 0 {
		t.Fatal("future register should be 0 after EndTransaction")
	}
}
