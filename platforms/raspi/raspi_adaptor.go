package raspi

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"
	"strconv"
	"strings"

	"github.com/hybridgroup/gobot"
	"github.com/hybridgroup/gobot/sysfs"
)

var _ gobot.AdaptorInterface = (*RaspiAdaptor)(nil)

var boardRevision = func() (string, string) {
	cat, _ := exec.Command("cat", "/proc/cpuinfo").Output()
	grep := exec.Command("grep", "Revision")
	in, _ := grep.StdinPipe()
	out, _ := grep.StdoutPipe()
	grep.Start()
	in.Write([]byte(string(cat)))
	in.Close()
	buf, _ := ioutil.ReadAll(out)
	grep.Wait()

	s := strings.Split(string(buf), " ")
	a := fmt.Sprintf("0x%v", strings.TrimSuffix(s[len(s)-1], "\n"))
	d, _ := strconv.ParseInt(a, 0, 64)

	rev := ""
	if d <= 3 {
		rev = "1"
	} else if d <= 15 {
		rev = "2"
	} else {
		rev = "3"
	}

	if rev == "1" {
		return rev, "/dev/i2c-0"
	}
	return rev, "/dev/i2c-1"
}

type RaspiAdaptor struct {
	gobot.Adaptor
	revision    string
	i2cLocation string
	digitalPins map[int]sysfs.DigitalPin
	i2cDevice   io.ReadWriteCloser
}

var pins = map[string]map[string]int{
	"3": map[string]int{
		"1": 0,
		"2": 2,
		"3": 2,
	},
	"5": map[string]int{
		"1": 1,
		"2": 3,
		"3": 3,
	},
	"7": map[string]int{
		"*": 4,
	},
	"8": map[string]int{
		"*": 14,
	},
	"10": map[string]int{
		"*": 15,
	},
	"11": map[string]int{
		"*": 17,
	},
	"12": map[string]int{
		"*": 18,
	},
	"13": map[string]int{
		"1": 21,
		"2": 27,
		"3": 27,
	},
	"15": map[string]int{
		"*": 22,
	},
	"16": map[string]int{
		"*": 23,
	},
	"18": map[string]int{
		"*": 24,
	},
	"19": map[string]int{
		"*": 10,
	},
	"21": map[string]int{
		"*": 9,
	},
	"22": map[string]int{
		"*": 25,
	},
	"23": map[string]int{
		"*": 11,
	},
	"24": map[string]int{
		"*": 8,
	},
	"26": map[string]int{
		"*": 7,
	},
	"29": map[string]int{
		"3": 5,
	},
	"31": map[string]int{
		"3": 6,
	},
	"32": map[string]int{
		"3": 12,
	},
	"33": map[string]int{
		"3": 13,
	},
	"35": map[string]int{
		"3": 19,
	},
	"36": map[string]int{
		"3": 16,
	},
	"37": map[string]int{
		"3": 26,
	},
	"38": map[string]int{
		"3": 20,
	},
	"40": map[string]int{
		"3": 21,
	},
}

// NewRaspiAdaptor creates a RaspiAdaptor with specified name and
func NewRaspiAdaptor(name string) *RaspiAdaptor {
	r := &RaspiAdaptor{
		Adaptor: *gobot.NewAdaptor(
			name,
			"RaspiAdaptor",
		),
		digitalPins: make(map[int]sysfs.DigitalPin),
	}
	rev, i2cLoc := boardRevision()
	r.revision = rev
	r.i2cLocation = i2cLoc
	return r
}

// Connect starts conection with board and creates
// digitalPins and pwmPins adaptor maps
func (r *RaspiAdaptor) Connect() error {
	return nil
}

// Finalize closes connection to board and pins
func (r *RaspiAdaptor) Finalize() error {
	for _, pin := range r.digitalPins {
		if pin != nil {
			pin.Unexport()
		}
	}
	if r.i2cDevice != nil {
		r.i2cDevice.Close()
	}
	return nil
}

// digitalPin returns matched digitalPin for specified values
func (r *RaspiAdaptor) digitalPin(pin string, dir string) (sysfsPin sysfs.DigitalPin, err error) {
	var i int

	if val, ok := pins[pin][r.revision]; ok {
		i = val
	} else if val, ok := pins[pin]["*"]; ok {
		i = val
	} else {
		err = errors.New("Not a valid pin")
		return
	}

	if r.digitalPins[i] == nil {
		r.digitalPins[i] = sysfs.NewDigitalPin(i)
		if err = r.digitalPins[i].Export(); err != nil {
			return
		}
	}

	if err = r.digitalPins[i].Direction(dir); err != nil {
		return
	}

	return r.digitalPins[i], nil
}

// DigitalRead reads digital value from pin
func (r *RaspiAdaptor) DigitalRead(pin string) (val int, err error) {
	sysfsPin, err := r.digitalPin(pin, sysfs.IN)
	if err != nil {
		return
	}
	return sysfsPin.Read()
}

// DigitalWrite writes digital value to specified pin
func (r *RaspiAdaptor) DigitalWrite(pin string, val byte) (err error) {
	sysfsPin, err := r.digitalPin(pin, sysfs.OUT)
	if err != nil {
		return err
	}
	return sysfsPin.Write(int(val))
}

// PwmWrite Not Implemented
func (r *RaspiAdaptor) PwmWrite(pin string, val byte) (err error) {
	return errors.New("PwmWrite is not yet implemented.")
}

// I2cStart starts a i2c device in specified address
func (r *RaspiAdaptor) I2cStart(address byte) (err error) {
	r.i2cDevice, err = sysfs.NewI2cDevice(r.i2cLocation, address)
	return err
}

// I2CWrite writes data to i2c device
func (r *RaspiAdaptor) I2cWrite(data []byte) (err error) {
	_, err = r.i2cDevice.Write(data)
	return
}

// I2cRead returns value from i2c device using specified size
func (r *RaspiAdaptor) I2cRead(size uint) (data []byte, err error) {
	data = make([]byte, size)
	_, err = r.i2cDevice.Read(data)
	return
}
