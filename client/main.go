package main

import (
	"fmt"

	"github.com/AlessandroBianchi/Peerster/client/localclient"
)

func main() {
	c := localclient.New()

	c.SetInfos()

	fmt.Println(c)

	c.ConnectToServer()
}
