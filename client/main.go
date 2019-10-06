package main

import (
	"github.com/AlessandroBianchi/Peerster/client/localclient"
)

func main() {
	c := localclient.New()

	c.SetInfos()

	c.ConnectToServer()
}
