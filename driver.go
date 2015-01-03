package main

import (
	"bufio"
	"os/exec"
	"regexp"
	"time"

	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/go-ninja/support"
)

var info = ninja.LoadModuleInfo("./package.json")
var log = logger.GetLogger(info.Name)

type Driver struct {
	support.DriverSupport
}

func NewDriver() (*Driver, error) {

	driver := &Driver{}

	err := driver.Init(info)
	if err != nil {
		log.Fatalf("Failed to initialize driver: %s", err)
	}

	err = driver.Export(driver)
	if err != nil {
		log.Fatalf("Failed to export driver: %s", err)
	}

	return driver, nil
}

var macRegex = regexp.MustCompile(`([a-fA-F0-9]{2}(?:|:)){6}`)

type DeviceListResponse struct {
	CmdResponse
	List []FoundDevice `json:"list"`
}

type FoundDevice struct {
	Name    string `json:"name"`
	Mac     string `json:"mac"`
	Netstat int    `json:"netstat"`
	New     int    `json:"new"`
	Lock    int    `json:"lock"`
	Type    int    `json:"type"`
}

//list":[{"name":"spmini","mac":"b4:43:0d:11:c2:04","netstat":1,"new":0,"lock":0,"type":10024},{"name":"MS1","mac":"cc:d2:9b:f5:60:54","netstat":1,"new":0,"lock":0,"type":10015}]

func (d *Driver) Start(_ interface{}) error {
	log.Infof("Driver Starting")

	go d.startServer()

	devices := make(map[string]*Device)

	go func() {

		for {
			time.Sleep(time.Second * 5)

			log.Debugf("Finding devices")

			var response DeviceListResponse

			err := cmd(&response, "12") // List devices

			if err != nil {
				log.Warningf("Failed to list devices: %s", err)
			}

			log.Debugf("Found %d devices", len(response.List))

			for _, found := range response.List {

				if _, ok := devices[found.Mac]; !ok {
					log.Infof("New name: %s mac: %s", found.Name, found.Mac)
					x, err := NewDevice(d, d.Conn, found.Name, found.Mac)
					if err != nil {
						log.Infof("Failed to create device: %s", err)
					} else {
						devices[found.Mac] = x
					}
				}
			}

		}

	}()

	return nil
}

func (d *Driver) startServer() {

	for {

		log.Infof("Starting server")
		cmd := exec.Command("./vsd-demo")

		reader, err := cmd.StdoutPipe()
		if err != nil {
			continue
		}

		bufReader := bufio.NewReader(reader)

		err = cmd.Start()

		if err != nil {
			continue
		}

		for {
			l, _, err := bufReader.ReadLine()

			if err != nil {
				log.Warningf("Server error: %s", err)
				break
			}

			log.Debugf("Server: %s", l)
		}

		time.Sleep(time.Second * 2)

	}

}
