/// Author: Bernhard Tittelbach, btittelbach@github  (c) 2015
package bbhw

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

var foundit_error_ error
var ERROR_DTO_ALREADY_LOADED error
var slots_file_ string

func init() {
	foundit_error_ = fmt.Errorf("Success")
	ERROR_DTO_ALREADY_LOADED = fmt.Errorf("DeviceTreeOverlay is already loaded")
}

func makeFindDirHelperFunc(returnvalue *string, path_base string, target_re, interm_re *regexp.Regexp) func(string, os.FileInfo, error) error {
	return func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			return nil
		}
		if path == path_base {
			return nil
		}
		if target_re.MatchString(path) {
			*returnvalue = path
			return foundit_error_ //foundit
		}
		if interm_re != nil && !interm_re.MatchString(path) {
			return filepath.SkipDir //skipdir if not like path_re1
		}
		return nil //continue walking
	}
}

// Singelton Function .. unlikely that slot file moves between reboots
func findSlotsFile() (sfile string, err error) {
	if len(slots_file_) > 0 {
		return slots_file_, nil
	}
	path_base := "/sys/devices"
	path_re1 := "^" + path_base + "/bone_capemgr" + `\.\d+`
	re1 := regexp.MustCompile(path_re1 + "$")
	var tdir string
	err = filepath.Walk(path_base, makeFindDirHelperFunc(&tdir, path_base, re1, nil))
	if err == foundit_error_ {
		err = nil
		sfile = tdir + "/slots"
	} else if err == nil {
		err = fmt.Errorf("OverlaySlotsFile Not Found")
	}
	slots_file_ = sfile
	return
}

func findOverlayStateFile(dtb_name string) (sfile string, err error) {
	path_base := "/sys/devices"
	path_re1 := "^" + path_base + "/ocp" + `\.\d+`
	path_re2 := path_re1 + "/" + dtb_name + `\.\d+`
	var re2 *regexp.Regexp
	re1 := regexp.MustCompile(path_re1 + "$")
	re2, err = regexp.Compile(path_re2)
	if err != nil {
		return
	}
	err = filepath.Walk(path_base, makeFindDirHelperFunc(&sfile, path_base, re2, re1))
	if err == foundit_error_ {
		err = nil
		sfile = sfile + "/state"
	} else if err == nil {
		err = fmt.Errorf("OverlayStateFile Not Found")
	}
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

func AddDeviceTreeOverlayIfNotAlreadyLoaded(dtb_name string) (err error) {
	slot, err := FindDeviceTreeOverlaySlot(dtb_name)
	if slot == -1 && err != nil {
		return AddDeviceTreeOverlay(dtb_name)
	} else {
		return ERROR_DTO_ALREADY_LOADED
	}
}

func FindDeviceTreeOverlaySlot(dtb_name string) (slotnum int64, err error) {
	var slotsfilename string
	var slotsfh *os.File
	var re *regexp.Regexp
	re, err = regexp.Compile(`^\s*(\d+): .*,` + dtb_name + "\n$")
	if err != nil {
		return
	}
	slotsfilename, err = findSlotsFile()
	if err != nil {
		return
	}
	slotsfh, err = os.OpenFile(slotsfilename, os.O_RDONLY|os.O_SYNC, 0666)
	if err != nil {
		return
	}
	defer slotsfh.Close()
	slotsfh.Seek(0, 0)
	slotsreader := bufio.NewReader(slotsfh)
	var line string
	for line, err = slotsreader.ReadString('\n'); err == nil; line, err = slotsreader.ReadString('\n') {
		if match := re.FindStringSubmatch(line); match != nil {
			slotnum, err = strconv.ParseInt(match[1], 10, 64)
			return
		}
	}
	return -1, fmt.Errorf("DeviceTreeOverlay %s not found in slots\n", dtb_name)
}

func RemoveDeviceTreeOverlay(dtb_name string) (err error) {
	var slotnum int64
	slotnum, err = FindDeviceTreeOverlaySlot(dtb_name)
	if err != nil {
		return err
	}
	var slotsfilename string
	var slotsfh *os.File
	slotsfilename, err = findSlotsFile()
	if err != nil {
		return
	}
	slotsfh, err = os.OpenFile(slotsfilename, os.O_RDWR|os.O_SYNC, 0666)
	if err != nil {
		return
	}
	defer slotsfh.Close()
	slotsfh.Truncate(0)
	slotsfh.WriteString(fmt.Sprintf("-%d\n", slotnum))
	return
}

// Overlays like PyBBIO-gpio.* can, once loaded, be configured using file /sys/devices/ocp.\d/PyBBIO-gpio.*.\d\d/state
// usually the following values can be written to configure the state:
// "mode_0b00101111" => INPUT, No Pullup/down
// "mode_0b00110111" => INPUT, Pullup
// "mode_0b00100111" => INPUT, Pulldown
// "mode_0b00001111" => OUTPUT, No Pullup/down
// "mode_0b00010111" => OUTPUT, Pullup
// "mode_0b00000111" => OUTPUT, Pulldown
func SetOverlayState(dtb_name, state string) (err error) {
	var overlaystatefile string
	var statefh *os.File
	overlaystatefile, err = findOverlayStateFile(dtb_name)
	if err != nil {
		return err
	}
	statefh, err = os.OpenFile(overlaystatefile, os.O_WRONLY|os.O_SYNC, 0666)
	if err != nil {
		return
	}
	defer statefh.Close()
	statefh.Truncate(0)
	statefh.WriteString(state)
	return
}
