package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
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

	err = mediaPlayer.EnableMediaChannel()
	if err != nil {
		mediaPlayer.Log().Fatalf("Failed to enable media channel: %s", err)
	}

	mediaPlayer.ApplyVolume = device.applyVolume
	if err := mediaPlayer.EnableVolumeChannel(true); err != nil {
		mediaPlayer.Log().Fatalf("Failed to enable volume channel: %s", err)
	}

	mediaPlayer.ApplyPlayPause = device.applyPlayPause
	if err := mediaPlayer.EnableControlChannel([]string{}); err != nil {
		mediaPlayer.Log().Fatalf("Failed to enable control channel: %s", err)
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

	if config.Bool(false, "togglePlay") {
		toggle := true
		go func() {
			for {
				toggle = !toggle

				err = device.applyPlayPause(toggle)

				if err != nil {
					mediaPlayer.Log().Warningf("Failed to set play/pause: %s", err)
				}
				time.Sleep(time.Second * 2)
			}
		}()
	}

	return device, nil
}

func (d *SpeakerDevice) updateState() error {

	var rsp ValueResponse

	err := d.vsd.Request(map[string]interface{}{
		"api_id":  429,
		"command": "ms1_request_pb",
		"mac":     d.id,
	}, &rsp)

	if err != nil {
		return err
	}

	spew.Dump("State response", rsp)

	var state map[string]interface{}

	idx := strings.Index(rsp.Value, "\n")

	err = json.Unmarshal([]byte(rsp.Value[idx:]), &state)
	if err != nil {
		return err
	}

	spew.Dump("State", state)

	name := state["name"].(string)

	track := &channels.MusicTrackMediaItem{
		ID:    &name,
		Title: &name,
	}

	err = d.device.UpdateMusicMediaState(track, nil)
	if err != nil {
		return fmt.Errorf("Failed sending media state: %s", err)
	}

	playState := state["status"].(string)

	switch playState {
	case "play":
		d.device.UpdateControlState(channels.MediaControlEventPlaying)
	case "pause":
		d.device.UpdateControlState(channels.MediaControlEventPaused)
	default:
		d.device.Log().Warningf("Unknown player status: %s", playState)
	}

	return nil

}

const keyPower = 2
const keyPlayPause = 3
const keyVolumeUp = 4
const keyVolumeDown = 5
const keyPause = 9

func (d *SpeakerDevice) applyPlayPause(play bool) error {

	var rsp CmdResponse
	var err error

	if play {
		//{"api_id":423,"command":"ms1_set_key_val", "mac":"cc:d2:9b:f5:61:b6", "value":1, "source":0}

		err = d.vsd.Request(map[string]interface{}{
			"api_id":  423,
			"command": "ms1_set_key_val",
			"mac":     d.id,
			"value":   1,
			"source":  0,
		}, &rsp)

		//spew.Dump("play response", rsp)

	} else {

		err = d.vsd.Request(Request{
			ApiID:   423,
			Command: "ms1_set_key_val",
			MAC:     d.id,
			Value:   keyPause,
		}, &rsp)

	}

	if err != nil {
		return err
	}

	//spew.Dump("pause response", rsp)
	return d.updateState()
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
