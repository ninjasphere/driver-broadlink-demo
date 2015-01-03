package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/devices"
	"github.com/ninjasphere/go-ninja/model"
)

type Device struct {
	device *devices.SwitchDevice
	id     string
}

func NewDevice(driver ninja.Driver, conn *ninja.Connection, id string) (*Device, error) {

	device := &Device{
		id: id,
	}

	switchDevice, err := devices.CreateSwitchDevice(driver, &model.Device{
		NaturalID:     id,
		NaturalIDType: "mdns",
		Name:          &id,
		Signatures: &map[string]string{
			"ninja:manufacturer": "Broadlink",
			"ninja:productName":  "SP-Mini",
			"ninja:thingType":    "switch", // (Sorry @thatguydan)
		},
	}, conn)

	if err != nil {
		return nil, err
	}

	device.device = switchDevice

	switchDevice.ApplyOnOff = device.applyOnOff

	toggle := true
	go func() {
		for {
			toggle = !toggle

			err = device.applyOnOff(toggle)

			if err != nil {
				switchDevice.Log().Warningf("Failed to set on/off: %s", err)
			}
			time.Sleep(time.Second * 2)
		}
	}()

	return device, nil
}

func (d *Device) applyOnOff(state bool) error {

	d.device.Log().Infof("applyonoff %t", state)

	s := "0"
	if state {
		s = "1"
	}

	var rsp CmdResponse
	err := cmd(&rsp, "84", d.id, s)

	if rsp.Msg == "FAILURE" {
		return fmt.Errorf("Failed to actuate device: %v", rsp)
	}

	spew.Dump(rsp)

	return err
}

type CmdResponse struct {
	Code   int    `json:"code"`
	Msg    string `json:"msg"`
	RetVal string `json:"retval"`
}

var responseRegex = regexp.MustCompile(`get response: (.*) len`)

func cmd(response interface{}, params ...string) error {

	log.Infof("Running command with %v", params)

	cmd := exec.Command("./demo-client", params...)

	output, err := cmd.Output()
	//log.Infof("Output from script: %s err:", output, err)

	if err != nil {
		return err
	}

	chunks := responseRegex.FindAllStringSubmatch(string(output), -1)

	if len(chunks) < 1 || len(chunks[0]) < 2 {
		return fmt.Errorf("Couldn't parse response: %s", output)
	}

	log.Debugf("Response: %s", chunks[0][1])

	return json.Unmarshal([]byte(chunks[0][1]), response)
}
