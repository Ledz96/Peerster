package gossippacket

import "github.com/AlessandroBianchi/Peerster/message"

//GossipPacket is a packet to send through gossiping. Contains a SimpleMessage.
type GossipPacket struct {
	Simple *message.SimpleMessage
}
