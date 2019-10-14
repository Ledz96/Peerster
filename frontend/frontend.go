package frontend

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"

	"github.com/AlessandroBianchi/Peerster/gossiper"
)

//Server is a struct that represents a frontend web server
type Server struct {
	port     string
	gossiper *gossiper.Gossiper
}

/*func YourHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Gorilla!\n"))
}*/

//New builds a new server
func New(g *gossiper.Gossiper) *Server {
	s := Server{port: g.ClientPort(), gossiper: g}
	return &s
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
	}
	//case POST: send message?
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
	}
	//case POST: add node
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
