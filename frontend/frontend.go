package frontend

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/dedis/protobuf"
	"github.com/gorilla/mux"

	"github.com/AlessandroBianchi/Peerster/gossiper"
	"github.com/AlessandroBianchi/Peerster/message"
)

//Server is a struct that represents a frontend web server
type Server struct {
	port     string
	gossiper *gossiper.Gossiper
	conn     *net.UDPConn
}

/*func YourHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Gorilla!\n"))
}*/

//New builds a new server
func New(g *gossiper.Gossiper) *Server {
	s := Server{port: g.ClientPort(), gossiper: g}

	udpAddr, err := net.ResolveUDPAddr("udp4", "127.0.0.1:"+s.port)
	if err != nil {
		fmt.Println("Error: could not resolve UDP address")
		os.Exit(-1)
	}
	conn, err := net.DialUDP("udp4", nil, udpAddr)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(-1)
	}
	s.conn = conn

	return &s
}

func (s Server) sendMessage(msg string) {
	msgStruct := message.Message{Text: msg}

	packetBytes, err := protobuf.Encode(&msgStruct)
	if err != nil {
		fmt.Printf("Client: error in encoding the message %v", err)
		os.Exit(-1)
	}

	n, err := s.conn.Write(packetBytes[0:])
	if err != nil {
		fmt.Printf("Error in sending the message. Error code: %v\n", err)
		os.Exit(-1)
	}
	fmt.Printf("%d bytes correctly sent!\n", n)
}

func (s Server) getLatestRumorMessagesHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		msgList := s.gossiper.GetLatestRumorMessagesList()
		msgListJSON, err := json.Marshal(msgList)
		if err != nil {
			fmt.Println("Error in marshaling the msgList")
			os.Exit(-1)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(msgListJSON)
	case "POST":
		err := r.ParseForm()
		if err != nil {
			fmt.Println("Error in reading infos from frontend")
			return
		}
		msg := r.FormValue("content")
		fmt.Println("From frontend, I read", msg)
		s.sendMessage(msg)
	}
}

func (s Server) getLatestNodesInfosHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		nodeList := s.gossiper.GetLatestNodesList()
		nodeListJSON, err := json.Marshal(nodeList)
		if err != nil {
			fmt.Println("Error in marshaling the msgList")
			os.Exit(-1)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(nodeListJSON)
	case "POST":
		err := r.ParseForm()
		if err != nil {
			fmt.Println("Error in reading infos from frontend")
			return
		}
		ip := r.FormValue("ip")
		port := r.FormValue("port")
		addr, err := net.ResolveUDPAddr("udp4", ip+":"+port)
		if err != nil {
			fmt.Println("Error in resolving the address")
			return
		}
		s.gossiper.AddPeer(addr)
	}
}

func (s Server) getPeerIDHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		id := s.gossiper.Name()
		idJSON, err := json.Marshal(id)
		if err != nil {
			fmt.Println("Error in marshaling the ID")
			os.Exit(-1)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(idJSON)
	case "PUT":

	}
}

//Start starts the server
func (s Server) Start() {
	r := mux.NewRouter()
	r.HandleFunc("/message", s.getLatestRumorMessagesHandler)
	r.HandleFunc("/node", s.getLatestNodesInfosHandler)
	r.HandleFunc("/id", s.getPeerIDHandler)
	r.Handle("/", http.FileServer(http.Dir("./frontend")))
	log.Fatal(http.ListenAndServe(":"+s.port, r))
	//	for {
	//		err := http.ListenAndServe(":"+s.port, nil)
	//		// error handling, etc..
	//		if err != nil {
	//			fmt.Println("Fatal error: can't listen")
	//		}
	//	}

	/*r := mux.NewRouter()
	// Routes consist of a path and a handler function.
	r.HandleFunc("/message", func(w http.ResponseWriter, r *http.Request) {
		// an example API handler

		json.NewEncoder(w).Encode()
	})
	r.Handle("/", http.FileServer(http.Dir(".")))

	// Bind to a port and pass our router in
	log.Fatal(http.ListenAndServe(":8080", r))
	*/
}
