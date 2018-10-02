/// Author: Bernhard Tittelbach, btittelbach@github  (c) 2018
package bbhw

import (
	"testing"
)

func Test_GetCPUInfo(t *testing.T) {
	i, err := GetCPUInfos()
	if err != nil {
		t.Error("Error getting CPUInfos", err)
	}
	t.Log(i)
	if i["processor"][0] != "0" {
		t.Error("Could not find expected info in CPUInfos")
	}
}
