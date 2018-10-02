/// Author: Bernhard Tittelbach, btittelbach@github  (c) 2018
package bbhw

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

/// return info from /proc/cpuinfo as map of keys to array of strings
/// e.g.
/// m["processor"]=["0","1"]
/// m["BogoMIPS"]=["38.40","38.40"]
/// m["Serial"]=["00000000001"]
func GetCPUInfos() (map[string][]string, error) {
	returninfos := make(map[string][]string)

	cpuinfo, err := os.OpenFile("/proc/cpuinfo", os.O_RDONLY|os.O_SYNC, 0666)
	if err != nil {
		return nil, fmt.Errorf("Could not open /proc/cpuinfo")
	}
	defer cpuinfo.Close()
	cpuinfo.Seek(0, 0)
	inforeader := bufio.NewReader(cpuinfo)
	var line string
	for line, err = inforeader.ReadString('\n'); err == nil; line, err = inforeader.ReadString('\n') {
		fields := strings.SplitN(line, ":", 2)
		if len(fields) == 2 {
			key := strings.TrimSpace(fields[0])
			value := strings.TrimSpace(fields[1])
			if pv, inmap := returninfos[key]; inmap {
				returninfos[key] = append(pv, value)
			} else {
				returninfos[key] = []string{value}
			}
		}
	}
	if err == io.EOF {
		err = nil
	}
	return returninfos, err
}
