package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

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
				log.Warningf("Failed to set on/off: %s", err)
			}
			time.Sleep(time.Second * 2)
		}
	}()

	return device, nil
}

func (d *Device) applyOnOff(state bool) error {

	s := "0"
	if state {
		s = "1"
	}

	cmd := exec.Command("./demo-client", d.id, "48", s)

	output, err := cmd.Output()
	log.Infof("Output from script: %s err:", output, err)

	if !strings.Contains(strings.ToLower(string(output)), "success") {
		return fmt.Errorf("Failed: %s", output)
	}

	return err

}
