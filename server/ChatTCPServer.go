/**
 * EasyTCPServer.go
 **/

// First Name: Hoang Giang
// Last Name: VO
// Student ID: 50211440

package main

import (
	"bytes"
	"container/list"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"
)

type serverStruct struct {
	serverPort      string
	timeStart       time.Time
	clients         *list.List
	channels        channelStruct
	commandsChan    commandStruct
	isClientAllowed chan bool
	nbClients       int
	listener        net.Listener
}

type channelStruct struct {
	newClient     chan clientStruct
	removeClient  chan clientStruct
	updateClient  chan clientStruct
	checkUsername chan []byte
}

type commandStruct struct {
	list    chan clientStruct
	dm      chan dmStruct
	exit    chan clientStruct
	ver     chan clientStruct
	rtt     chan clientStruct
	message chan clientStruct
	ihp     chan clientStruct
}

type dmStruct struct {
	originClient    clientStruct
	destinationName string
	message         []byte
}

type clientStruct struct {
	clientResponse       []byte
	clientResponseLength int
	clientIp             string
	clientPort           string
	clientRequestCounter int
	clientConnection     net.Conn
	clientId             int
	clientUsername       []byte
	serverRunTime        time.Time
	serverChan           channelStruct
	serverCmdChan        commandStruct
}

func command1(client clientStruct) {
	client.serverCmdChan.list <- client
}

func command2(client clientStruct) {
	// fmt.Println(client.clientResponse)
	// client.serverCmdChan.dm <- dm
}

func command3(client clientStruct) {
	client.serverChan.removeClient <- client
}

func command4(client clientStruct) {
	client.serverCmdChan.ver <- client
}

func command5(client clientStruct) {
	client.serverCmdChan.rtt <- client
}

func splitIpPort(client *clientStruct, clientIpPort string) {
	delimiterIndex := 0
	for index := len(clientIpPort) - 1; index >= 0; index-- {
		if clientIpPort[index] == ':' {
			delimiterIndex = index
			break
		}
	}
	client.clientPort = clientIpPort[delimiterIndex+1 : len(clientIpPort)]
	client.clientIp = clientIpPort[0:delimiterIndex]
}

// Initializing the struct server and start listening to incoming clients
func startServer() (*serverStruct, net.Listener, error) {
	server := &serverStruct{
		timeStart:  time.Now(),
		serverPort: strconv.Itoa((50211440 % 10000) + 20000),
		clients:    list.New(),
		channels: channelStruct{
			newClient:     make(chan clientStruct),
			removeClient:  make(chan clientStruct),
			updateClient:  make(chan clientStruct),
			checkUsername: make(chan []byte, 1024),
		},
		commandsChan: commandStruct{
			list:    make(chan clientStruct),
			dm:      make(chan dmStruct),
			exit:    make(chan clientStruct),
			ver:     make(chan clientStruct),
			rtt:     make(chan clientStruct),
			message: make(chan clientStruct),
			ihp:     make(chan clientStruct),
		},
		isClientAllowed: make(chan bool),
	}

	listener, err := net.Listen("tcp", ":"+server.serverPort)
	if err != nil {
		return server, nil, err
	}
	server.listener = listener
	fmt.Printf("The server is ready to receive on port %s\n\n", server.serverPort)

	return server, listener, nil
}

// Function running in a goroutine in order to handle
// concurrent task when manipulating the list of clients such as
// removing, adding, or update clients for example
func concurrentTaskServer(server serverStruct) {
	for {
		select {
		case newClient := <-server.channels.newClient:
			server.clients.PushFront(newClient)
			newClient.clientConnection.Write([]byte("[Welcome " + string(newClient.clientUsername) + " to CAU network class chat room at " + server.listener.Addr().String() + ".] \n[There are " + strconv.Itoa(server.clients.Len()) + " users connected.]"))
			fmt.Println(string(newClient.clientUsername) + " joined from " + newClient.clientIp + ":" + newClient.clientPort + ". There are " + strconv.Itoa(server.clients.Len()) + " users connected\n")
			break
		case clientToBeRemoved := <-server.channels.removeClient:

			for i := server.clients.Front(); i != nil; {
				currentElement := i.Value.(clientStruct)
				nextElement := i.Next()
				if currentElement.clientId == clientToBeRemoved.clientId {
					currentElement.clientConnection.Write([]byte("[" + string(currentElement.clientUsername) + " left. There are " + strconv.Itoa(server.clients.Len() - 1) + " users now]\n\n"))
					currentElement.clientConnection.Close()
					server.clients.Remove(i)
					break
				}
				i = nextElement
			}
			for b := server.clients.Front(); b != nil; {
				currentElement := b.Value.(clientStruct)
				currentElement.clientConnection.Write([]byte("[" + string(clientToBeRemoved.clientUsername) + " left. There are " + strconv.Itoa(server.clients.Len()) + " users now]\n\n"))
				b = b.Next()
			}
			fmt.Println(string(clientToBeRemoved.clientUsername) + " left. There are " + strconv.Itoa(server.clients.Len()) + " users now\n")
		case clientToBeUpdated := <-server.channels.updateClient:
			for i := server.clients.Front(); i != nil; {
				currentElement := i.Value.(clientStruct)
				if currentElement.clientId == clientToBeUpdated.clientId {
					i.Value = clientToBeUpdated
				}
				i = i.Next()
			}
			break
		case username := <-server.channels.checkUsername:
			for i := server.clients.Front(); i != nil; {
				currentElement := i.Value.(clientStruct)
				if bytes.Compare(currentElement.clientUsername, username) == 0 {
					server.isClientAllowed <- false
					break
				}
				i = i.Next()
			}
			server.isClientAllowed <- true
			break
		}
	}
}

func removeZeros(btar []byte) []byte {
	count := 0
	for i := 0; i < len(btar); i++ {
		if btar[i] != 0 {
			count++
		}
	}
	tmpArray := make([]byte, count)
	index := 0
	for b := 0; b < len(btar); b++ {
		if btar[b] != 0 {
			tmpArray[index] = btar[b]
			index++
		}
	}
	return tmpArray
}

// Similar to the function running in a goroutine to handle concurrent task on the list of clients
// This function will handle concurrent task among the commands that the client will send to the server
func concurrentCmdServer(server serverStruct) {
	for {
		select {
		case client := <-server.commandsChan.message:
			for i := server.clients.Front(); i != nil; {
				currentElement := i.Value.(clientStruct)
				if bytes.Equal(currentElement.clientUsername, client.clientUsername) == false {
					currentElement.clientConnection.Write([]byte(string(client.clientUsername) + "> " + string(client.clientResponse) + "\n\n"))
				}
				i = i.Next()
			}
		case client := <-server.commandsChan.list:
			listString := string("")
			for i := server.clients.Front(); i != nil; {
				currentElement := i.Value.(clientStruct)
				if i.Next() == nil {
					listString += string(currentElement.clientUsername) + "," + string(currentElement.clientIp) + "," + string(currentElement.clientPort) + "\n\n"
				} else {
					listString += string(currentElement.clientUsername) + "," + string(currentElement.clientIp) + "," + string(currentElement.clientPort) + "\n"
				}
				i = i.Next()
			}
			client.clientConnection.Write([]byte(listString))
		case dmClient := <-server.commandsChan.dm:
			for i := server.clients.Front(); i != nil; {
				currentElement := i.Value.(clientStruct)
				if bytes.Compare(removeZeros(currentElement.clientUsername), []byte(dmClient.destinationName)) == 0 {
					currentElement.clientConnection.Write([]byte("from: " + string(dmClient.originClient.clientUsername) + "> " + string(dmClient.message) + "\n\n"))
					break
				}
				i = i.Next()
			}
		case client := <-server.commandsChan.ver:
			client.clientConnection.Write([]byte("Server's software version: 0.1\n\n"))
		case client := <-server.commandsChan.rtt:
			client.clientConnection.Write([]byte("rtt"))
		case client := <-server.commandsChan.ihp:
			for b := server.clients.Front(); b != nil; {
				currentElement := b.Value.(clientStruct)
				if currentElement.clientId != client.clientId {
					currentElement.clientConnection.Write([]byte(string(client.clientUsername) + "> " + string(client.clientResponse) + "\n"))
				}
				b = b.Next()
			}
			for i := server.clients.Front(); i != nil; {
				currentElement := i.Value.(clientStruct)
				nextElement := i.Next()
				if currentElement.clientId == client.clientId {
					client.clientConnection.Write([]byte("[You have been disconnected from the server.]\n[Reason: You hate the professor.]\n\n"))
					currentElement.clientConnection.Close()
					server.clients.Remove(i)
				}
				i = nextElement
			}
			for b := server.clients.Front(); b != nil; {
				currentElement := b.Value.(clientStruct)
				currentElement.clientConnection.Write([]byte("[" + string(client.clientUsername) + " is disconnected. " + "There are " + strconv.Itoa(server.clients.Len()) + " users in the chat room.]\n\n"))
				b = b.Next()
			}
			fmt.Println("[" + string(client.clientUsername) + " is disconnected. " + "There are " + strconv.Itoa(server.clients.Len()) + " users in the chat room.]\n")
		}
	}
}

func signalInterrupt(server serverStruct, listener net.Listener) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			if sig == os.Interrupt {
				listener.Close()
				os.Exit(0)
			}
		}
	}()
}

// Server loop that will handle all incoming connection
// It will check for the username and if the server is full
// If there are no issues, the client will be created
func serverLoop(server serverStruct, listener net.Listener) {
	clientId := 0
	for true {
		// Accepting an incoming connection
		conn, err := listener.Accept()
		if err != nil {
			continue
		}

		if (server.clients.Len() + 1 > 8) {
			conn.Write([]byte("chatting room full. cannot connect"))
			conn.Close()
			continue
		}

		clientUsername := make([]byte, 1024)
		_, _ = conn.Read(clientUsername)

		server.channels.checkUsername <- clientUsername

		t := <-server.isClientAllowed

		if t == false {
			conn.Close()
			continue
		}

		clientId += 1
		// Creating a new client structure
		newClient := &clientStruct{
			clientResponse:       make([]byte, 0),
			clientResponseLength: 0,
			clientIp:             "",
			clientPort:           "",
			clientRequestCounter: 0,
			clientConnection:     conn,
			clientId:             clientId,
			serverRunTime:        server.timeStart,
			serverChan:           server.channels,
			serverCmdChan:        server.commandsChan,
			clientUsername:       clientUsername,
		}
		// Sending a channel in order to push a new client structure
		// into our list of clients
		splitIpPort(newClient, newClient.clientConnection.RemoteAddr().String())
		newClient.serverChan.newClient <- *newClient
		// Goroutine to handle the client's commands
		go clientThread(*newClient)
	}
}

func main() {
	// Initializing server
	server, listener, error := startServer()
	// Checks if the initialization had any errors, if yes we exit with error code 84
	if error != nil {
		fmt.Println(error.Error())
		os.Exit(84)
	}

	// Goroutine in order to handle all
	// tasks related to the list of client
	// (push new client from list, remove client from list, update client)
	// with channels so that It's concurrent
	go concurrentTaskServer(*server)
	// Same as above but it handles the client's commands
	go concurrentCmdServer(*server)
	// Goroutine to exit gracefully when there is an ctrl+c input
	signalInterrupt(*server, listener)

	serverLoop(*server, listener)

}

// The client's thread that is running
// in a goroutine once a client is created server side
func clientThread(client clientStruct) {
	fncPtr := map[byte]func(clientStruct){
		'1': command1,
		'2': command2,
		'3': command3,
		'4': command4,
		'5': command5,
	}

	err := *new(error)
	for {
		client.clientResponse = make([]byte, 10024)
		client.clientResponseLength, err = client.clientConnection.Read(client.clientResponse)
		client.clientRequestCounter += 1
		client.clientResponse = client.clientResponse[:len(client.clientResponse)-1]

		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}

		if client.clientResponse[0] == '2' {
			command := strings.Split(string(client.clientResponse), " ")
			message := string("")
			for i := 2; i < len(command); i++ {
				if i == 2 {
					message += command[i]
				} else {
					message += " " + command[i]
				}
			}
			newDm := dmStruct{
				originClient:    client,
				destinationName: command[1],
				message:         []byte(message),
			}
			client.serverCmdChan.dm <- newDm
			continue
		}

		if client.clientResponseLength == 1 && client.clientResponse[0] >= '1' && client.clientResponse[0] <= '5' {
			fncPtr[client.clientResponse[0]](client)
		} else if client.clientResponseLength == 1 && (client.clientResponse[0] < '1' || client.clientResponse[0] > '5') {
			fmt.Printf("invalid command\n\n")
		} else {
			if strings.Contains(strings.ToLower(string(client.clientResponse)), "i hate professor") == true {
				client.serverCmdChan.ihp <- client
				break
			}
			client.serverCmdChan.message <- client
		}

	}
}
