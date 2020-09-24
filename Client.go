package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

var conn net.Conn

type NetworksList struct {
	networkMap map[string]net.Conn
	mux        sync.Mutex
}

type MapOfStrings struct {
	mapOS map[string]bool
	mux   sync.Mutex
}

// Send message to all connected channels
func BroadCast(inc chan string, list *NetworksList) {
	for {
		msg := <-inc
		list.mux.Lock()
		for _, k := range list.networkMap {
			k.Write([]byte(msg))
		}
		list.mux.Unlock()
	}
}

// Recieve from all connected channels
func Recieve(channels chan string, list *NetworksList, mapOfStrings *MapOfStrings, conn net.Conn) {
	for {
		msg, err := bufio.NewReader(conn).ReadString('\n')

		if err != nil {
			list.mux.Lock()
			fmt.Println("Ending session with " + conn.RemoteAddr().String())
			fmt.Print("> ")
			delete(list.networkMap, conn.RemoteAddr().String())
			list.mux.Unlock()
			break
		}
		mapOfStrings.mux.Lock()
		if mapOfStrings.mapOS[msg] != true {
			fmt.Print("Recieved String: " + string(msg))
			fmt.Print("> ")
			channels <- msg
			fmt.Println("String added to saved messages: " + msg)
			fmt.Print("> ")
			mapOfStrings.mapOS[msg] = true
		}
		mapOfStrings.mux.Unlock()
	}

}

// ###OPTIONAL EXERCISE### Sends all previous strings in the P2P network to the listener
func SendPreviousStrings(InputConn net.Conn, mapOfStrings *MapOfStrings) {
	for i := range mapOfStrings.mapOS {
		InputConn.Write([]byte(i))
		time.Sleep(2 * time.Millisecond)
	}
}

// Handler for each connection connected
func HandleConnections(InputConn net.Conn, list *NetworksList, channels chan string, mapOfStrings *MapOfStrings) {
	defer InputConn.Close()

	// ### OPTIONAL EXERCISE ###
	go SendPreviousStrings(InputConn, mapOfStrings)

	Recieve(channels, list, mapOfStrings, InputConn)
}

// Handler resposible for looking after connections
func LookForConnection(ln net.Listener, list *NetworksList, channels chan string, mapOfStrings *MapOfStrings) {
	defer ln.Close()

	go BroadCast(channels, list)

	for {
		fmt.Println("Listening for connection...")
		fmt.Print("> ")
		InputConn, _ := ln.Accept()
		fmt.Println("Got a connection...")
		fmt.Print("> ")
		list.mux.Lock()
		list.networkMap[InputConn.RemoteAddr().String()] = InputConn
		list.mux.Unlock()
		go HandleConnections(InputConn, list, channels, mapOfStrings)
	}
}

// Responsible for letting the client send messages to all connections
func SendManuallyToConnections(channels chan string, mapOfStrings *MapOfStrings) {
	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("> ")
		text, err := reader.ReadString('\n')
		mapOfStrings.mux.Lock()
		mapOfStrings.mapOS[text] = true
		mapOfStrings.mux.Unlock()

		if err != nil {
			return
		}
		channels <- text
	}

}

func main() {
	list := new(NetworksList)
	list.networkMap = make(map[string]net.Conn)
	channels := make(chan string)
	mapOfStrings := new(MapOfStrings)
	mapOfStrings.mapOS = make(map[string]bool)

	fmt.Println("Write ip-address: ")
	var ipInput string
	fmt.Print("> ")
	fmt.Scan(&ipInput)

	fmt.Println("Write port: ")
	var portInput string
	fmt.Print("> ")
	fmt.Scan(&portInput)

	var ipPort string
	ipPort = ipInput + ":" + portInput
	conn, err := net.Dial("tcp", ipPort)

	if err != nil {
		ln, _ := net.Listen("tcp", ":18081")
		defer ln.Close()
		for {
			fmt.Println("Local Ip-Address and port number: " + "127.0.0.1:18081")
			go LookForConnection(ln, list, channels, mapOfStrings)

			SendManuallyToConnections(channels, mapOfStrings)
		}
	}

	defer conn.Close()
	LocalIPPort := conn.LocalAddr().String()
	fmt.Println("Local Ip-Address and port number: " + LocalIPPort)

	s := strings.Split(LocalIPPort, ":")
	port := s[1]

	list.mux.Lock()
	list.networkMap[conn.RemoteAddr().String()] = conn
	list.mux.Unlock()

	ln, _ := net.Listen("tcp", ":"+port)

	go LookForConnection(ln, list, channels, mapOfStrings)

	go Recieve(channels, list, mapOfStrings, conn)

	SendManuallyToConnections(channels, mapOfStrings)
}
