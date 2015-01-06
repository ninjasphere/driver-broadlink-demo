package main

import (
	"fmt"
	"time"

	"github.com/ninjasphere/driver-broadlink-demo/vsd"
	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/channels"
	"github.com/ninjasphere/go-ninja/config"
	"github.com/ninjasphere/go-ninja/devices"
	"github.com/ninjasphere/go-ninja/model"
)

type SpeakerDevice struct {
	device *devices.MediaPlayerDevice
	id     string
	vsd    *vsd.Connection
}

func NewSpeakerDevice(driver ninja.Driver, conn *ninja.Connection, vsd *vsd.Connection, name, id string) (*SpeakerDevice, error) {

	device := &SpeakerDevice{
		id:  id,
		vsd: vsd,
	}

	mediaPlayer, err := devices.CreateMediaPlayerDevice(driver, &model.Device{
		NaturalID:     id,
		NaturalIDType: "mdns",
		Name:          &name,
		Signatures: &map[string]string{
			"ninja:manufacturer": "Broadlink",
			"ninja:productName":  "MS1",
			"ninja:thingType":    "mediaplayer", // (Sorry @thatguydan)
		},
	}, conn)

	if err != nil {
		return nil, err
	}

	device.device = mediaPlayer

	mediaPlayer.ApplyVolume = device.applyVolume
	if err := mediaPlayer.EnableVolumeChannel(true); err != nil {
		mediaPlayer.Log().Fatalf("Failed to enable volume channel: %s", err)
	}

	if config.Bool(false, "volume") {
		toggle := 0.0
		go func() {
			for {
				err = device.applyVolume(&channels.VolumeState{
					Level: &toggle,
				})

				if err != nil {
					mediaPlayer.Log().Warningf("Failed to set volume: %s", err)
				}
				time.Sleep(time.Second * 1)

				toggle += 0.1
				if toggle > 1 {
					toggle = 0
				}
			}
		}()
	}

	return device, nil
}

// {"api_id":420,"command":"ms1_set_vol", "mac":"cc:d2:9b:f5:61:b6", "value":4}
func (d *SpeakerDevice) applyVolume(state *channels.VolumeState) error {
	d.device.Log().Infof("applyVolume: %v", state)

	var rsp CmdResponse

	err := d.vsd.Request(Request{
		ApiID:   420,
		Command: "ms1_set_vol",
		MAC:     d.id,
		Value:   int(*state.Level * 10),
	}, &rsp)

	if err != nil {
		return fmt.Errorf("Failed to actuate device: %s", err)
	}

	if rsp.Message == "FAILURE" {
		return fmt.Errorf("Failed to actuate device: %v", rsp)
	}

	return nil
}
