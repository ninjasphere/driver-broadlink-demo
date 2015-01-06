package main

import "github.com/ninjasphere/go-ninja/support"

func main() {

	_, err := NewDriver()

	if err != nil {
		log.Errorf("Failed to create driver: %s", err)
		return
	}

	support.WaitUntilSignal()
}
