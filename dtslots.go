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
var dtsslot_slots_file_ string
var dtsslot_ocp_dir_ string = ""

var dtsslot_path_ocp_regex_ *regexp.Regexp = regexp.MustCompile("^/sys/devices(?:/platform)?/ocp" + `(?:\.\d+)?`)
var dtsslot_path_base_ string = "/sys/devices"

func init() {
	foundit_error_ = fmt.Errorf("Success")
	ERROR_DTO_ALREADY_LOADED = fmt.Errorf("DeviceTreeOverlay is already loaded")
}

func findFile(basedir, searchdirregex, searchfilename string, maxdepth int) (sfile string, err error) {
	var dir_regex *regexp.Regexp
	if dir_regex, err = regexp.Compile(filepath.Join(basedir, searchdirregex)); err != nil {
		return "", err
	}
	err = filepath.Walk(basedir, makeFindDirHelperFunc(&sfile, dir_regex, maxdepth))
	if err == foundit_error_ {
		err = nil
	} else if err == nil {
		err = fmt.Errorf("Directory Not Found: %s/%s", basedir, searchdirregex)
		return
	}
	sfile = filepath.Join(sfile, searchfilename)
	if _, err = os.Stat(sfile); os.IsNotExist(err) {
		return "", err
	}
	return
}

func makeFindDirHelperFunc(returnvalue *string, target_re *regexp.Regexp, maxdepth int) func(string, os.FileInfo, error) error {
	return func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			return nil
		}
		if len(filepath.SplitList(path)) > maxdepth {
			return filepath.SkipDir
		}
		if target_re.MatchString(path) {
			*returnvalue = path
			return foundit_error_ //foundit
		}
		return nil //continue walking
	}
}

// Singelton Function .. unlikely that slot file moves between reboots
func findSlotsFile() (sfile string, err error) {
	if len(dtsslot_slots_file_) > 0 {
		return dtsslot_slots_file_, nil
	}

	sfile, err = findFile(dtsslot_path_base_, "(?:platform)?/bone_capemgr"+`(?:\.\d+)?`+"$", "slots", 5)

	if err != nil {
		err = fmt.Errorf("OverlaySlotsFile Not Found")
		sfile = ""
		return
	}
	dtsslot_slots_file_ = sfile
	return
}

func findOCPDir() (path string, err error) {
	if len(dtsslot_ocp_dir_) > 0 {
		return dtsslot_ocp_dir_, nil
	} else {

		err = filepath.Walk(dtsslot_path_base_, makeFindDirHelperFunc(&path, dtsslot_path_ocp_regex_, 6))
		if err == foundit_error_ && len(path) > 0 {
			dtsslot_ocp_dir_ = path
			err = nil
			return
		} else if err == nil {
			err = fmt.Errorf("OCP directory not found")
		}
	}
	return
}

func findOverlayStateFile(dtb_name string) (sfile string, err error) {
	var ocp_dir string
	if ocp_dir, err = findOCPDir(); err != nil {
		return
	}

	sfile, err = findFile(ocp_dir, "(?:ocp:)?"+dtb_name+"(?:_pinmux)?", "state", 7)

	if err != nil {
		err = fmt.Errorf("Overlay state file not found")
		sfile = ""
		return
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

// Newer Universal Overlays allow the pinmux to be set by writing
// "gpio", "pwm", "default", "spi", "i2c", .. to the state file for a certain pin
//
// while overlays like PyBBIO-gpio.* can, once loaded, be configured using file /sys/devices/ocp.\d/PyBBIO-gpio.*.\d\d/state
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
