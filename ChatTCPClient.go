/**
 * EasyTCPClient.go
 **/

// First Name: Hoang Giang
// Last Name: VO
// Student ID: 50211440

package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"
)

type clientStruct struct {
	stdin []byte
	lastCommand []byte
	serverConnection net.Conn
	clock time.Time
	server string
	port string
	username string
}

func startClient() (*clientStruct, error) {
	client := &clientStruct {
		stdin: []byte(""),
		serverConnection: nil,
		clock: time.Now(),
		server: "localhost",
		port: strconv.Itoa((50211440 % 10000) + 20000),
		username: os.Args[1],
	}

	var err error
	client.serverConnection, err = net.Dial("tcp", client.server+":"+client.port)

	if err != nil {
		return client, err
	}

	// localAddr := client.serverConnection.LocalAddr().(*net.TCPAddr)
	// fmt.Printf("The client is running on port %d\n", localAddr.Port)
	return client, nil
}

// Function to read from the server the first time we connect
// if we are allowed to connect to the chat room
// that implies knowing if the server is full or if our username is not a duplicata
func firstHandshake(client *clientStruct) {
	count, err := client.serverConnection.Write([]byte(os.Args[1]))
	if err != nil {
		os.Exit(0)
	}
	if count <= 0 {
		os.Exit(0)
	}
	if count > 0 {
		buffer := make([]byte, 3064)
		count, err := client.serverConnection.Read(buffer)
		if err != nil {
			fmt.Println("\n[that nickname is already used by another user. cannot connect.]")
			os.Exit(0)
		}
		if count <= 0 {
			fmt.Println("\n[that nickname is already used by another user. cannot connect.]")
			os.Exit(0)
		}
		if count > 0 {
			fmt.Println()
			fmt.Println(string(buffer) + "\n");
		}
	}
}

// Switch case to recognize commands or invalid commands
func getCommand(command []string, client *clientStruct) []byte {
	switch string(command[0]) {
		case "\\list":
			return []byte{'1'}
		case "\\dm":
			message := string("")
			for i := 2; i < len(command); i++ {
				if i == 2 {
					message += command[i]
				} else {
					message += " " + command[i]
				}
			}
			serverMsg := []byte("2" + " " + command[1] + " " + message)
			return serverMsg
		case "\\exit":
			return []byte{'3'}
		case "\\ver":
			return []byte{'4'}
		case "\\rtt":
			return []byte{'5'}
		default:
			return []byte{'9'}
	}
}

func readFromServer(client *clientStruct, buffer []byte) {
	count, err := client.serverConnection.Read(buffer)
	if err != nil {
		client.serverConnection.Close()
		os.Exit(0)
	}
	if count <= 0 {
		client.serverConnection.Close()
		os.Exit(0)
	}
	if count > 0 {
		if strings.Compare(string(client.lastCommand), "\\rtt") == 0 {
			rtt := time.Since(client.clock).Seconds() * 1000
			fmt.Printf("RTT = %.3f ms\n\n", rtt)
		} else {
			fmt.Printf("%s", string(buffer))
			if strings.Compare(string(client.lastCommand), "\\exit") == 0 {
				fmt.Println("gg~")
			}
		}
	}
}

func writeToServer(client *clientStruct, buffer []byte) {
	count, err := client.serverConnection.Write(buffer)
	if err != nil {
		client.serverConnection.Close()
		os.Exit(0)
	}
	if count <= 0 {
		client.serverConnection.Close()
		os.Exit(0)
	}
	fmt.Println()
}

func clientLoop(client *clientStruct) {
	for {
		client.stdin = make([]byte, 10024)

		client.stdin, _ = bufio.NewReader(os.Stdin).ReadBytes('\n')
		client.stdin = client.stdin[:len(client.stdin) - 1]

		client.lastCommand = client.stdin

		if len(client.stdin) > 0 && client.stdin[0] == '\\' {
			commandSplit := strings.Split(string(client.stdin), " ")
			result := getCommand(commandSplit, client)
			if result[0] == '9' {
				fmt.Println("invalid command")
				writeToServer(client, result)
				continue
			}
			client.clock = time.Now()
			writeToServer(client, result)
		} else if len(client.stdin) > 0 {
			writeToServer(client, client.stdin)
		}
	}
}

// Checking if the username is alpha
// and if it's less than 32 characters
func checkUsername(s string) bool {
    for _, r := range s {
        if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
			return false
		}
    }
	if (len(s) > 32) {
		return false
	}
    return true
}

func main() {
	if (len(os.Args) < 2 || len(os.Args) > 2 || !checkUsername(os.Args[1])) {
		// If there are no arguments
		// we will exit the program with the error code 84
		fmt.Printf("Username is incorrect. Please enter an English nickname in order to enter the chatroom.")
		fmt.Println("\nHow to use: go run ChatTCPClient yourusername")
		os.Exit(0);
	}
	// Initializing client connecting to server
	client, err := startClient()

	// Error handling of startClient
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(0)
	}

	// Goroutine to handle CTRL+C signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			if sig == os.Interrupt {
				client.serverConnection.Write([]byte{'3'});
				client.lastCommand = []byte("\\exit")
			}
		}
	}()

	// Reading from the server in a goroutine
	go func() {
		buffer := make([]byte, 10024)

		for {
			buffer = make([]byte, 10024)
			readFromServer(client, buffer)
		}
	}()

	firstHandshake(client)
	clientLoop(client)
	os.Exit(0)
}
