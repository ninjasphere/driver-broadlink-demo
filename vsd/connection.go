package vsd

import (
	"encoding/json"
	"fmt"
	"io"
	"net"

	"github.com/ninjasphere/go-ninja/logger"
)

var log = logger.GetLogger("test")

type cmd struct {
	request  interface{}
	response interface{}
	done     chan error
}

type Connection struct {
	incoming chan cmd
}

func (c *Connection) Request(request interface{}, response interface{}) error {
	x := cmd{
		request:  request,
		response: response,
		done:     make(chan error, 1),
	}

	c.incoming <- x
	err := <-x.done
	return err
}

func Connect(host string) (*Connection, error) {

	conn := &Connection{
		incoming: make(chan cmd, 16),
	}

	// Set up the outgoing connection
	out, err := net.Dial("udp", host+":9040")
	if err != nil {
		return nil, fmt.Errorf("Failed to open outgoing connection: %s", err)
	}
	enc := json.NewEncoder(out)

	// Set up the incoming connection
	addr := net.UDPAddr{
		Port: 9050,
		IP:   net.ParseIP("127.0.0.1"),
	}
	in, err := net.ListenUDP("udp", &addr)
	if err != nil {
		return nil, fmt.Errorf("Failed to open incoming connection: %s", err)
	}
	dec := json.NewDecoder(in)

	go func() {
		for cmd := range conn.incoming {

			if err := enc.Encode(cmd.request); err != nil {
				cmd.done <- err
				continue
			}

			if err := dec.Decode(&cmd.response); err == io.EOF {
				log.Fatalf("EOF!", err)
			} else if err != nil {
				cmd.done <- err
				continue
			}

			cmd.done <- nil
		}
	}()

	return conn, nil
}
