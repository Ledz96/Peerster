package message

//SimpleMessage is a structure that contains original sender's name, addr (as ip:port) of the relay peer and content of the message
type SimpleMessage struct {
	OriginalName  string
	RelayPeerAddr string
	Contents      string
}
