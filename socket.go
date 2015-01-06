package main

import (
	"fmt"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ninjasphere/driver-broadlink-demo/vsd"
	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/config"
	"github.com/ninjasphere/go-ninja/devices"
	"github.com/ninjasphere/go-ninja/model"
)

type SocketDevice struct {
	device *devices.SwitchDevice
	id     string
	vsd    *vsd.Connection
}

func NewSocketDevice(driver ninja.Driver, conn *ninja.Connection, vsd *vsd.Connection, name, id string) (*SocketDevice, error) {

	device := &SocketDevice{
		id:  id,
		vsd: vsd,
	}

	switchSocketDevice, err := devices.CreateSwitchDevice(driver, &model.Device{
		NaturalID:     id,
		NaturalIDType: "mdns",
		Name:          &name,
		Signatures: &map[string]string{
			"ninja:manufacturer": "Broadlink",
			"ninja:productName":  "SP-Mini",
			"ninja:thingType":    "socket", // (Sorry @thatguydan)
		},
	}, conn)

	if err != nil {
		return nil, err
	}

	device.device = switchSocketDevice

	switchSocketDevice.ApplyOnOff = device.applyOnOff

	if config.Bool(false, "toggle") {
		toggle := true
		go func() {
			for {
				toggle = !toggle

				err = device.applyOnOff(toggle)

				if err != nil {
					switchSocketDevice.Log().Warningf("Failed to set on/off: %s", err)
				}
				time.Sleep(time.Second * 2)
			}
		}()
	}

	return device, nil
}

func (d *SocketDevice) applyOnOff(state bool) error {

	d.device.Log().Infof("applyonoff %t", state)

	s := 0
	if state {
		s = 1
	}

	var rsp CmdResponse

	err := d.vsd.Request(Request{
		ApiID:   84,
		Command: "switch_operation",
		MAC:     "b4:43:0d:95:ef:c1",
		Index:   &s,
	}, &rsp)

	if err != nil {
		return fmt.Errorf("Failed to actuate device: %s", err)
	}

	if rsp.Message == "FAILURE" {
		return fmt.Errorf("Failed to actuate device: %v", rsp)
	}

	spew.Dump(rsp)

	return err
}
