// (c) Bernhard Tittelbach, 2013

package bbhw

import (
	"bufio"
	"errors"
	"io"
	"os"
	"syscall"
	"unsafe"
)

// ---------- Termios Code ----------------
func SetRawFd(ttyfd uintptr) (syscall.Termios, error) {
	var orig_termios syscall.Termios
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(ttyfd), uintptr(syscall.TCGETS), uintptr(unsafe.Pointer(&orig_termios)))
	if errno != 0 {
		return orig_termios, os.NewSyscallError("SYS_IOCTL", errno)
	}
	new_termios := orig_termios
	new_termios.Iflag &= ^uint32(syscall.BRKINT | syscall.INLCR | syscall.ICRNL | syscall.IGNCR | syscall.INPCK | syscall.ISTRIP | syscall.IXOFF | syscall.IXON)
	new_termios.Oflag &= ^uint32(syscall.OPOST)
	new_termios.Lflag &= ^uint32(syscall.ECHO | syscall.ECHONL | syscall.ICANON | syscall.IEXTEN | syscall.ISIG)
	new_termios.Cflag |= uint32(syscall.CS8)

	new_termios.Cc[syscall.VMIN] = 1
	new_termios.Cc[syscall.VTIME] = 0

	if err := SetTermiosFd(new_termios, ttyfd); err != nil {
		return orig_termios, err
	}
	return orig_termios, nil
}

func SetRawFile(f *os.File) (syscall.Termios, error) {
	return SetRawFd(f.Fd())
}

func SetTermiosFd(termios syscall.Termios, ttyfd uintptr) error {
	r1, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(ttyfd), uintptr(syscall.TCSETS), uintptr(unsafe.Pointer(&termios)))
	if errno != 0 {
		return os.NewSyscallError("SYS_IOCTL", errno)
	}
	if r1 != 0 {
		return errors.New("Error during ioctl tcsets syscall")
	}
	return nil
}

func SetSpeedFd(ttyfd uintptr, speed uint32) (err error) {
	var orig_termios syscall.Termios
	r1, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(ttyfd), uintptr(syscall.TCGETS), uintptr(unsafe.Pointer(&orig_termios)))
	if errno != 0 {
		return os.NewSyscallError("SYS_IOCTL", errno)
	}

	//~ orig_termios.Ispeed = speed
	//~ orig_termios.Ospeed = speed
	//input baudrate == output baudrate and we ignore special case B0
	//orig_termios.Cflag &= ^(syscall.CBAUD | syscall.CBAUDEX)

	//for x86
	orig_termios.Cflag |= speed
	//for mips etc.
	orig_termios.Ispeed = speed
	orig_termios.Ospeed = speed

	r1, _, errno = syscall.Syscall(syscall.SYS_IOCTL, uintptr(ttyfd), uintptr(syscall.TCSETS), uintptr(unsafe.Pointer(&orig_termios)))
	if errno != 0 {
		return os.NewSyscallError("SYS_IOCTL", errno)
	}
	if r1 != 0 {
		return errors.New("Error during ioctl tcsets syscall")
	}
	return nil
}

func SetSpeedFile(f *os.File, speed uint32) error {
	return SetSpeedFd(f.Fd(), speed)
}

// ---------- Serial TTY Code -------------
func openTTY(name string, speed uint) (file *os.File, err error) {
	file, err = os.OpenFile(name, os.O_RDWR, 0666)
	if err != nil {
		return
	}
	if _, err = SetRawFile(file); err != nil {
		return
	}
	switch speed {
	case 0: // set no baudrate
	case 1200:
		err = SetSpeedFile(file, syscall.B1200)
	case 2400:
		err = SetSpeedFile(file, syscall.B2400)
	case 4800:
		err = SetSpeedFile(file, syscall.B4800)
	case 9600:
		err = SetSpeedFile(file, syscall.B9600)
	case 19200:
		err = SetSpeedFile(file, syscall.B19200)
	case 38400:
		err = SetSpeedFile(file, syscall.B38400)
	case 57600:
		err = SetSpeedFile(file, syscall.B57600)
	case 115200:
		err = SetSpeedFile(file, syscall.B115200)
	case 230400:
		err = SetSpeedFile(file, syscall.B230400)
	default:
		file.Close()
		err = errors.New("Unsupported Baudrate, use 0 to disable setting a baudrate")
	}
	return
}

func serialWriter(in <-chan string, serial *os.File) {
	for totty := range in {
		serial.WriteString(totty)
		serial.Sync()
	}
	serial.Close()
}

// advantage of using linescanner is, that it matches for \r?\n
// i.e. any possible occuring \r is automatically stripped
func serialReaderLineScanner(out chan<- string, serial *os.File) {
	linescanner := bufio.NewScanner(serial)
	linescanner.Split(bufio.ScanLines)
	for linescanner.Scan() {
		if err := linescanner.Err(); err != nil {
			panic(err.Error())
		}
		text := linescanner.Text()
		if len(text) == 0 {
			continue
		}
		out <- text
	}
}

//for other cases, i.e. strange devices that terminate it's output
//with '\r' (e.g. for those that cat /dev/ttyO1 should output values but not fill the screen)
//we cannot use LineScanner and thus have this nice little functions
func serialReaderDelim(out chan<- string, serial *os.File, delim byte) {
	rd := bufio.NewReader(serial)
	for {
		//readstring returns string INCLUDING delimintor
		text, err := rd.ReadString(delim)
		if err == io.EOF {
			close(out)
			return
		}
		if err != nil {
			panic(err.Error())
		}
		if len(text) <= 1 {
			continue
		}
		out <- text[0 : len(text)-1]
	}
}

func OpenAndHandleSerial(filename string, serspeed uint) (chan string, chan string, error) {
	serial, err := openTTY(filename, serspeed)
	if err != nil {
		return nil, nil, err
	}
	wr := make(chan string, 1)
	rd := make(chan string, 20)
	go serialWriter(wr, serial)
	go serialReaderLineScanner(rd, serial)
	return wr, rd, nil
}

func OpenAndHandleStrangeSerial(filename string, serspeed uint, delim byte) (chan string, chan string, error) {
	serial, err := openTTY(filename, serspeed)
	if err != nil {
		return nil, nil, err
	}
	wr := make(chan string, 1)
	rd := make(chan string, 20)
	go serialWriter(wr, serial)
	go serialReaderDelim(rd, serial, delim)
	return wr, rd, nil
}
