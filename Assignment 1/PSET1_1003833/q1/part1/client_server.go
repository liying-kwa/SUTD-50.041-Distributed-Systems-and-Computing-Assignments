package main

import (
	"fmt"
	"math/rand"
	"time"
)

const NUM_CLIENTS int = 5
const NUM_MESSAGES int = 3
const MESSAGE_INTERVAL = 500

type Server struct {
	channel     chan Message
	clientArray []Client
}

type Client struct {
	clientID      int
	clientChannel chan Message
	server        Server
	closeChannel  chan int
}

type Message struct {
	senderID  int
	messageID int
}

func (s Server) listen() {

	fmt.Println("Server is listening...")
	numDoneClients := 0

	for {

		// Receive message
		clientMessage := <-s.channel
		fmt.Printf("Server has received Message %d.%d \n", clientMessage.senderID, clientMessage.messageID)

		// Broadcast message
		for _, c := range s.clientArray {
			if c.clientID == clientMessage.senderID {
				continue
			} else {
				serverMessage := Message{clientMessage.senderID, clientMessage.messageID}
				go s.sendMessage(c.clientChannel, serverMessage)
			}
		}

		// If the last message is received, close server channel
		if clientMessage.messageID == NUM_MESSAGES {
			numDoneClients += 1
			if numDoneClients == NUM_CLIENTS {
				fmt.Println("Server received all messaages. Stopping Server...")
				return
			}
		}

		// Randomly delay next received message while broadcasting
		amt := time.Duration(rand.Intn(1000))
		time.Sleep(time.Millisecond * amt)

	}
}

func (s Server) sendMessage(clientChannel chan Message, serverMessage Message) {
	clientChannel <- serverMessage
}

func (c Client) sendMessages() {

	for i := 1; i <= NUM_MESSAGES; i++ {

		// Sleep for a period of time before sending message
		time.Sleep(time.Millisecond * MESSAGE_INTERVAL)

		// Construct clientMessage
		clientMessage := Message{c.clientID, i}

		// Send clientMessage to server
		fmt.Printf("Client %d is sending Message %d.%d \n", c.clientID, c.clientID, clientMessage.messageID)
		c.server.channel <- clientMessage

	}
}

func (c Client) listen() {

	for {

		select {

		// Receive message from server
		case broadcastedMessage := <-c.clientChannel:
			fmt.Printf("Client %d has received Message %d.%d \n", c.clientID, broadcastedMessage.senderID, broadcastedMessage.messageID)

		// Close channel because server has broadcasted all messages
		case <-c.closeChannel:
			fmt.Println("Client", c.clientID, "is closing...")
			return

		default:

		}

	}

}

func main() {

	// Create server
	channel := make(chan Message)
	clientArray := []Client{}
	s := Server{channel, clientArray}

	// Create clients
	for i := 1; i <= NUM_CLIENTS; i++ {
		clientChannel := make(chan Message)
		closeChannel := make(chan int)
		c := Client{i, clientChannel, s, closeChannel}
		s.clientArray = append(s.clientArray, c)
	}

	// Each client listens for messages from server
	// Each client also sends a message to the server periodically
	for _, c := range s.clientArray {
		go c.sendMessages()
		go c.listen()
	}

	// Server listens for messages from any client
	// Server broadcasts each message to all other clients
	s.listen()

	// Server is done, but sleep for a while
	// to receive remaining msgs before stopping all clients
	time.Sleep(time.Second)
	for _, c := range s.clientArray {
		c.closeChannel <- 1
	}

	// Sleep for a while to wait for channels to close gracefully
	time.Sleep(time.Second)

}
