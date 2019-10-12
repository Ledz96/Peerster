package message

//Message is a structure that contains a message, without any additional information
type Message struct {
	Text string
}

//SimpleMessage is a structure that contains original sender's name, addr (as ip:port) of the relay peer and content of the message
type SimpleMessage struct {
	OriginalName  string
	RelayPeerAddr string
	Contents      string
}

//RumorMessage is a structure that contains the text of a rumor message to be gossiped. It contains information about the original sender and a sequence number (local to the sender)
type RumorMessage struct {
	Origin string
	ID     uint32
	Text   string
}

//PeerStatus is a structure that summarizes the set of messages a peer has seen. The NextID field represents the sequence number of the first unseen message from the source Identifier
type PeerStatus struct {
	Identifier string
	NextID     uint32
}

//StatusPacket is a packet that contains information about all the needed message for a peer
type StatusPacket struct {
	Want []PeerStatus
}

const (
	//Simple identifies a simple message
	Simple int = 1
	//Rumor identifies a rumor message
	Rumor int = 2
	//Status identifies a status packet
	Status int = 3
)

//Type is an array that identifies possible types for messages
var Type = []int{Simple, Rumor, Status}
