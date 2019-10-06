package localclient

import (
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/AlessandroBianchi/Peerster/message"
	"github.com/dedis/protobuf"
)

const buffsize int = 1024

type client struct {
	udpAddr *net.UDPAddr
	udpConn *net.UDPConn
	msg     message.SimpleMessage
}

//New creates a new client
func New() *client {
	c := client{}
	return &c
}

//SetInfos read infos from command line and stores them in the structure
func (c *client) SetInfos() {
	c.udpAddr, c.msg.Contents = getInfosFromCL()
}

func (c *client) ConnectToServer() {
	conn, err := net.DialUDP("udp4", nil, c.udpAddr)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(-1)
	}
	c.udpConn = conn

	packetBytes, err := protobuf.Encode(&c.msg)
	if err != nil {
		fmt.Println("Error in encoding the message")
		os.Exit(-1)
	}

	n, err := c.udpConn.Write(packetBytes[0:])
	if err != nil {
		fmt.Printf("Error in sending the message. Error code: %v\n", err)
		os.Exit(-1)
	}
	fmt.Printf("%d bytes correctly sent!\n", n)
}

/*
func (c *client) ListenToServer() {
	buffer := make([]byte, buffsize)

	n, err := bufio.NewReader(c.udpConn).Read(buffer)
	if err != nil {
		fmt.Println("Error in reading data from server")
		os.Exit(-1)
	}

	var msg *message.SimpleMessage

	if n > 0 {
		protobuf.Decode(buffer, msg)
	} else {
		fmt.Println("Couldn't read anything...")
	}
}*/

func getInfosFromCL() (*net.UDPAddr, string) {
	port := flag.String("UIPort", "8080", "Client port")
	msg := flag.String("msg", "", "Message that the client must send")

	flag.Parse()

	udpAddr, err := net.ResolveUDPAddr("udp4", "127.0.0.1:"+*port)
	if err != nil {
		fmt.Println("Error: could not resolve UDP address")
		os.Exit(-1)
	}

	return udpAddr, *msg
}
