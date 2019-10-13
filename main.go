package main

import (
	"github.com/AlessandroBianchi/Peerster/gossiper"
)

func main() {
	g := gossiper.New()

	g.SetInfos()

	if g.Simple() {
		go g.SimpleHandleClientMessages()
		g.SimpleHandlePeersMessages()
	} else {
		go g.HandleClientMessages()
		if g.IsAntiEntropyActive() {
			go g.AntiEntropy()
		}
		g.HandlePeersMessages()
	}
}
