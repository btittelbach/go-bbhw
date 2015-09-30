package bbhw

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

var foundit_error_ error

func init() {
	foundit_error_ = fmt.Errorf("Success")
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

func findSlotsFile() (sfile string, err error) {
	foundit := fmt.Errorf("Success")
	path_base := "/sys/devices"
	path_re1 := "^" + path_base + "/bone_capemgr" + `\.\d+`
	re1 := regexp.MustCompile(path_re1 + "$")
	var tdir string
	findDeviceTreeSlotsFileBBB := func(path string, info os.FileInfo, err error) error {
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

func findOverlayStateFile(dtb_name string) (sfile string, err error) {
	foundit := fmt.Errorf("Success")
	path_base := "/sys/devices"
	path_re1 := "^" + path_base + "/ocp" + `\.\d+`
	path_re2 := path_re1 + "/" + dtb_name + `\.\d+`
	re1 := regexp.MustCompile(path_re1 + "$")
	re2, err := regexp.Compile(path_re2)
	if err != nil {
		return
	}
	findOverlayStateFileBBB := func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			return nil
		}
		if path == path_base {
			return nil
		}
		if re2.MatchString(path) {
			sfile = path
			return foundit //foundit
		}
		if !re1.MatchString(path) {
			return filepath.SkipDir //skipdir if not like path_re1
		}
		return nil //continue walking
	}
	err = filepath.Walk(path_base, findOverlayStateFileBBB)
	if err == foundit {
		err = nil
	} else {
		err = fmt.Errorf("NotFound")
	}
	sfile = sfile + "/state"
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
