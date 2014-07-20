package bbhw

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

func findSlotsFile() (sfile string, err error) {
	foundit := fmt.Errorf("Success")
	path_base := "/sys/devices"
	path_re1 := "^" + path_base + "/bone_capemgr" + `\.\d+`
	re1 := regexp.MustCompile(path_re1 + "$")
	var tdir string
	findDeviceTreeSlotsFileBBB := func(path string, info os.FileInfo, err error) error {
		LogPins_.Print(path)
		if !info.IsDir() {
			return nil
		}
		if path == path_base {
			return nil
		}
		if re1.MatchString(path) {
			tdir = path
			return foundit //foundit
		}
		return nil //continue walking
	}
	err = filepath.Walk(path_base, findDeviceTreeSlotsFileBBB)
	if err == foundit {
		err = nil
	}
	sfile = tdir + "/slots"
	return
}

func AddDeviceTreeOverlay(dtb_name string) (err error) {
	var slotsfilename string
	var slotsfh *os.File
	slotsfilename, err = findSlotsFile()
	if err != nil {
		return
	}
	slotsfh, err = os.OpenFile(slotsfilename, os.O_WRONLY|os.O_SYNC, 0666)
	if err != nil {
		return
	}
	defer slotsfh.Close()
	slotsfh.Truncate(0)
	slotsfh.WriteString(dtb_name)
	return
}

func RemoveDeviceTreeOverlay(dtb_name string) (err error) {
	var slotsfilename string
	var slotsfh *os.File
	var re *regexp.Regexp
	re, err = regexp.Compile("^([0-9]+): .*," + dtb_name + "\n$")
	if err != nil {
		return
	}
	slotsfilename, err = findSlotsFile()
	if err != nil {
		return
	}
	slotsfh, err = os.OpenFile(slotsfilename, os.O_RDWR|os.O_SYNC, 0666)
	if err != nil {
		return
	}
	defer slotsfh.Close()
	slotsfh.Seek(0, 0)
	slotsreader := bufio.NewReader(slotsfh)
	var line string
	for line, err = slotsreader.ReadString('\n'); err == nil; line, err = slotsreader.ReadString('\n') {
		if match := re.FindStringSubmatch(line); match != nil {
			slotsfh.Truncate(0)
			slotsfh.WriteString(fmt.Sprintf("-%s\n", match[1]))
			return nil
		}
	}
	return fmt.Errorf("DeviceTreeOverlay %s not found in slots\n", dtb_name)
}
