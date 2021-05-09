package main

import (
	"fmt"
	"math/rand"
	"sort"
	"time"
)

const NUM_CLIENTS int = 3
const NUM_MESSAGES int = 2
const NUM_EVENTS int = (2 + 2*(NUM_CLIENTS-1)) * NUM_CLIENTS * NUM_MESSAGES
const MESSAGE_INTERVAL = 500

type Server struct {
	channel     chan Message
	clientArray []Client
	logicalTS   int
}

type Client struct {
	clientID      int
	clientChannel chan Message
	server        Server
	readyChannel  chan int
	closeChannel  chan int
	logicalTS     int
}

type Message struct {
	senderID  int
	messageID int
	logicalTS int
}

type Event struct {
	eventType  EventType
	senderID   int
	receiverID int
	messageID  int
	logicalTS  int
}

type EventType int

const (
	clientSending EventType = iota
	serverReceiving
	serverBroadcasting
	clientReceiving
)

func max(x int, y int) int {
	if x > y {
		return x
	}
	return y
}

func (s Server) listen(eventsChannel chan Event) {

	fmt.Println("Server is listening...")
	numDoneClients := 0

	for {

		// Receive message
		clientMessage := <-s.channel
		s.logicalTS = max(s.logicalTS, clientMessage.logicalTS) + 1
		fmt.Printf("[LogicalTS: %d] Server has received Message %d.%d \n", s.logicalTS, clientMessage.senderID, clientMessage.messageID)
		e := Event{serverReceiving, clientMessage.senderID, 0, clientMessage.messageID, s.logicalTS}
		eventsChannel <- e

		// Broadcast message
		s.logicalTS += 1
		for receiverID := 1; receiverID <= NUM_CLIENTS; receiverID++ {
			if receiverID == clientMessage.senderID {
				continue
			} else {
				serverMessage := Message{clientMessage.senderID, clientMessage.messageID, s.logicalTS}
				go s.sendMessage(eventsChannel, receiverID, serverMessage)
			}
		}

		// If the last message is received, close server channel
		if clientMessage.messageID == NUM_MESSAGES {
			numDoneClients += 1
			if numDoneClients == NUM_CLIENTS {
				fmt.Println("Server received all messaages. Stopping listening on Server...")
				return
			}
		}

		// Randomly delay next received message while broadcasting
		amt := time.Duration(rand.Intn(1000))
		time.Sleep(time.Millisecond * amt)

	}
}

func (s Server) sendMessage(eventsChannel chan Event, receiverID int, serverMessage Message) {
	fmt.Printf("[LogicalTS: %d] Server is sending Message %d.%d to Client %d \n", serverMessage.logicalTS, serverMessage.senderID, serverMessage.messageID, receiverID)
	s.clientArray[receiverID-1].clientChannel <- serverMessage
	e := Event{serverBroadcasting, serverMessage.senderID, receiverID, serverMessage.messageID, serverMessage.logicalTS}
	eventsChannel <- e
}

func (c Client) prepareMessages() {

	for i := 1; i <= NUM_MESSAGES; i++ {

		// Sleep for a period of time before sending message
		time.Sleep(time.Millisecond * MESSAGE_INTERVAL)

		// Prepare to send message in ready channel
		c.readyChannel <- i

	}

}

func (c Client) listenAndSend(eventsChannel chan Event) {

	for {

		select {

		// Send message to server
		case messageID := <-c.readyChannel:
			c.logicalTS += 1
			clientMessage := Message{c.clientID, messageID, c.logicalTS}
			fmt.Printf("[LogicalTS: %d] Client %d is sending Message %d.%d \n", c.logicalTS, c.clientID, clientMessage.senderID, clientMessage.messageID)
			c.server.channel <- clientMessage
			e := Event{clientSending, clientMessage.senderID, 0, clientMessage.messageID, clientMessage.logicalTS}
			eventsChannel <- e

		// Receive message from server
		case broadcastedMessage := <-c.clientChannel:
			c.logicalTS = max(c.logicalTS, broadcastedMessage.logicalTS) + 1
			fmt.Printf("[LogicalTS: %d] Client %d has received Message %d.%d \n", c.logicalTS, c.clientID, broadcastedMessage.senderID, broadcastedMessage.messageID)
			e := Event{clientReceiving, broadcastedMessage.senderID, c.clientID, broadcastedMessage.messageID, c.logicalTS}
			eventsChannel <- e

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
	s := Server{channel, clientArray, 0}

	// Create clients
	for i := 1; i <= NUM_CLIENTS; i++ {
		clientChannel := make(chan Message)
		readyChannel := make(chan int)
		closeChannel := make(chan int)
		c := Client{i, clientChannel, s, readyChannel, closeChannel, 0}
		s.clientArray = append(s.clientArray, c)
	}

	// Create events channel for Total Order later
	var eventsChannel chan Event = make(chan Event, NUM_EVENTS)

	// Each client listens for messages from server
	// Each client also sends a message to the server periodically
	for _, c := range s.clientArray {
		go c.prepareMessages()
		go c.listenAndSend(eventsChannel)
	}

	// Server listens for messages from any client
	// Server broadcasts each message to all other clients
	s.listen(eventsChannel)

	// Server is done, but sleep for a while
	// to receive remaining msgs before stopping all clients
	time.Sleep(time.Second)
	for _, c := range s.clientArray {
		c.closeChannel <- 1
	}

	// Sleep for a while to wait for channels to close gracefully
	time.Sleep(time.Second)

	// Print Total Order
	close(eventsChannel)
	eventsArray := []Event{}
	for e := range eventsChannel {
		eventsArray = append(eventsArray, e)
	}
	sort.Slice(eventsArray, func(i, j int) bool {
		return eventsArray[i].logicalTS < eventsArray[j].logicalTS
	})
	fmt.Println()
	time.Sleep(time.Second)
	fmt.Println("===================================")
	fmt.Println("            Total Order            ")
	fmt.Println("===================================")
	for i, e := range eventsArray {
		if e.eventType == clientSending {
			fmt.Printf("%d) [LogicalTS: %d] Client %d sends Message %d.%d to server \n", i+1, e.logicalTS, e.senderID, e.senderID, e.messageID)
		} else if e.eventType == serverReceiving {
			fmt.Printf("%d) [LogicalTS: %d] Server receives Message %d.%d \n", i+1, e.logicalTS, e.senderID, e.messageID)
		} else if e.eventType == serverBroadcasting {
			fmt.Printf("%d) [LogicalTS: %d] Server sends Message %d.%d to Client %d \n", i+1, e.logicalTS, e.senderID, e.messageID, e.receiverID)
		} else {
			fmt.Printf("%d) [LogicalTS: %d] Client %d receives Message %d.%d \n", i+1, e.logicalTS, e.receiverID, e.senderID, e.messageID)
		}
	}

}
