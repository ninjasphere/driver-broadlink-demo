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

func (d *Driver) Start(_ interface{}) error {
	log.Infof("Driver Starting")

	go d.startServer()

	devices := make(map[string]*Device)

	go func() {

		for {
			time.Sleep(time.Second * 5)

			log.Debugf("Finding devices")

			cmd := exec.Command("./demo-client", "list-or-whatever")

			output, err := cmd.Output()
			if err != nil {
				log.Warningf("Failed to list devices: %s", err)
			}

			//output = []byte("hello 01:23:45:67:89:ab \n aa:23:45:67:89:ab ,sdhshd sdjhbsd \n ff:23:45:67:89:ab ")

			for _, mac := range macRegex.FindAllString(string(output), -1) {
				log.Infof("Found mac: %s", mac)

				if _, ok := devices[mac]; !ok {
					log.Infof("New mac: %s", mac)
					x, err := NewDevice(d, d.Conn, mac)
					if err != nil {
						log.Infof("Failed to create device: %s", err)
					} else {
						devices[mac] = x
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
