package main

import (
	"fmt"

	"github.com/AlessandroBianchi/Peerster/client/client"
)

func main() {
	c := client.New()

	c.SetInfos()

	fmt.Println(c)

	c.ConnectToServer()
}
