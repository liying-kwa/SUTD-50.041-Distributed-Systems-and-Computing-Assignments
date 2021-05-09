package main

import (
	"fmt"
	"math/rand"
	"time"
)

const NUM_CLIENTS int = 3
const NUM_PROCESSORS int = NUM_CLIENTS + 1
const NUM_MESSAGES int = 2
const NUM_EVENTS int = (2 + 2*(NUM_CLIENTS-1)) * NUM_CLIENTS * NUM_MESSAGES
const MESSAGE_INTERVAL = 500

type Server struct {
	pid         int
	channel     chan Message
	clientArray []Client
	vectorClock [NUM_PROCESSORS]int
}

type Client struct {
	pid           int
	clientChannel chan Message
	server        Server
	readyChannel  chan int
	closeChannel  chan int
	vectorClock   [NUM_PROCESSORS]int
}

type Message struct {
	senderID    int
	messageID   int
	vectorClock [NUM_PROCESSORS]int
}

type Event struct {
	eventType   EventType
	senderID    int
	receiverID  int
	messageID   int
	vectorClock [NUM_PROCESSORS]int
}

type EventType int

const (
	clientSending EventType = iota
	serverReceiving
	serverBroadcasting
	clientReceiving
)

func maxInt(x int, y int) int {
	if x > y {
		return x
	}
	return y
}

func moreThanVC(v1 [NUM_PROCESSORS]int, v2 [NUM_PROCESSORS]int) bool {
	for i, _ := range v1 {
		if v1[i] > v2[i] {
			return true
		}
	}
	return false
}

func lessThanVC(v1 [NUM_PROCESSORS]int, v2 [NUM_PROCESSORS]int) bool {
	toReturn := false
	for i, _ := range v1 {
		// Each element is less than or equal
		if v1[i] > v2[i] {
			return false
		}
		// At least one element is less than
		if v1[i] < v2[i] {
			toReturn = true
		}
	}
	return toReturn
}

func mergeVC(v1 [NUM_PROCESSORS]int, v2 [NUM_PROCESSORS]int, receiverID int) [NUM_PROCESSORS]int {
	v3 := [NUM_PROCESSORS]int{}
	for i, _ := range v1 {
		v3[i] = maxInt(v1[i], v2[i])
	}
	v3[receiverID] += 1
	return v3
}

func (s Server) listen(eventsChannel chan Event, pcvChannel chan string) {

	fmt.Println("Server is listening...")
	numDoneClients := 0

	for {

		// Receive message
		clientMessage := <-s.channel
		if moreThanVC(s.vectorClock, clientMessage.vectorClock) {
			// Check for potential causality violation
			pcv := fmt.Sprintf("[PCV] Server received Message %d.%d. Local CV: %d; Message VC: %d", clientMessage.senderID, clientMessage.messageID, s.vectorClock, clientMessage.vectorClock)
			fmt.Println(pcv)
			pcvChannel <- pcv
		}
		s.vectorClock = mergeVC(s.vectorClock, clientMessage.vectorClock, s.pid)
		fmt.Printf("[vectorClock: %d] Server has received Message %d.%d \n", s.vectorClock, clientMessage.senderID, clientMessage.messageID)
		e := Event{serverReceiving, clientMessage.senderID, 0, clientMessage.messageID, s.vectorClock}
		eventsChannel <- e

		// Broadcast message
		s.vectorClock[s.pid] += 1
		for receiverID := 1; receiverID <= NUM_CLIENTS; receiverID++ {
			if receiverID == clientMessage.senderID {
				continue
			} else {
				serverMessage := Message{clientMessage.senderID, clientMessage.messageID, s.vectorClock}
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
	fmt.Printf("[vectorClock: %d] Server is sending Message %d.%d to Client %d \n", serverMessage.vectorClock, serverMessage.senderID, serverMessage.messageID, receiverID)
	s.clientArray[receiverID-1].clientChannel <- serverMessage
	e := Event{serverBroadcasting, serverMessage.senderID, receiverID, serverMessage.messageID, serverMessage.vectorClock}
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

func (c Client) listenAndSend(eventsChannel chan Event, pcvChannel chan string) {

	for {

		select {

		// Send message to server
		case messageID := <-c.readyChannel:
			c.vectorClock[c.pid] += 1
			clientMessage := Message{c.pid, messageID, c.vectorClock}
			fmt.Printf("[vectorClock: %d] Client %d is sending Message %d.%d \n", c.vectorClock, c.pid, clientMessage.senderID, clientMessage.messageID)
			c.server.channel <- clientMessage
			e := Event{clientSending, clientMessage.senderID, 0, clientMessage.messageID, clientMessage.vectorClock}
			eventsChannel <- e

		// Receive message from server
		case broadcastedMessage := <-c.clientChannel:
			if moreThanVC(c.vectorClock, broadcastedMessage.vectorClock) {
				// Check for potential causality violation
				pcv := fmt.Sprintf("[PCV] Client %d received Message %d.%d. Local CV: %d; Message VC: %d", c.pid, broadcastedMessage.senderID, broadcastedMessage.messageID, c.vectorClock, broadcastedMessage.vectorClock)
				fmt.Println(pcv)
				pcvChannel <- pcv
			}
			c.vectorClock = mergeVC(c.vectorClock, broadcastedMessage.vectorClock, c.pid)
			fmt.Printf("[vectorClock: %d] Client %d has received Message %d.%d \n", c.vectorClock, c.pid, broadcastedMessage.senderID, broadcastedMessage.messageID)
			e := Event{clientReceiving, broadcastedMessage.senderID, c.pid, broadcastedMessage.messageID, c.vectorClock}
			eventsChannel <- e
			// Check for potential causality violation

		// Close channel because server has broadcasted all messages
		case <-c.closeChannel:
			fmt.Println("Client", c.pid, "is closing...")
			return

		default:

		}

	}

}

func main() {

	// Create server with pid 0
	channel := make(chan Message)
	clientArray := []Client{}
	vectorClock := [NUM_PROCESSORS]int{}
	s := Server{0, channel, clientArray, vectorClock}

	// Create clients with pid 1 to NUM_CLIENTS
	for i := 1; i <= NUM_CLIENTS; i++ {
		clientChannel := make(chan Message)
		readyChannel := make(chan int)
		closeChannel := make(chan int)
		vectorClock := [NUM_PROCESSORS]int{}
		c := Client{i, clientChannel, s, readyChannel, closeChannel, vectorClock}
		s.clientArray = append(s.clientArray, c)
	}

	// Create events channel for Total Order later
	var eventsChannel chan Event = make(chan Event, NUM_EVENTS)

	// Create channel to detect potential causality violation
	var pcvChannel chan string = make(chan string, NUM_EVENTS)

	// Each client listens for messages from server
	// Each client also sends a message to the server periodically
	for _, c := range s.clientArray {
		go c.prepareMessages()
		go c.listenAndSend(eventsChannel, pcvChannel)
	}

	// Server listens for messages from any client
	// Server broadcasts each message to all other clients
	s.listen(eventsChannel, pcvChannel)

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
	/* sort.Slice(eventsArray, func(i, j int) bool {
		return lessThanVC(eventsArray[i].vectorClock, eventsArray[j].vectorClock)
	})
	fmt.Println("\n~~~~~~~~~~~~~TOTAL ORDER~~~~~~~~~~~~~~")
	for i, e := range eventsArray {
		if e.eventType == clientSending {
			fmt.Printf("%d) [vectorClock: %d] Client %d sends Message %d.%d to server \n", i+1, e.vectorClock, e.senderID, e.senderID, e.messageID)
		} else if e.eventType == serverReceiving {
			fmt.Printf("%d) [vectorClock: %d] Server receives Message %d.%d \n", i+1, e.vectorClock, e.senderID, e.messageID)
		} else if e.eventType == serverBroadcasting {
			fmt.Printf("%d) [vectorClock: %d] Server sends Message %d.%d to Client %d \n", i+1, e.vectorClock, e.senderID, e.messageID, e.receiverID)
		} else {
			fmt.Printf("%d) [vectorClock: %d] Client %d receives Message %d.%d \n", i+1, e.vectorClock, e.receiverID, e.senderID, e.messageID)
		}
	} */

	// Print events that can potentially cause causality violation
	close(pcvChannel)
	pcvArray := []string{}
	for pcv := range pcvChannel {
		pcvArray = append(pcvArray, pcv)
	}

	fmt.Println()
	time.Sleep(time.Second)
	fmt.Println("===========================================")
	fmt.Println("       Potential Causality Violation       ")
	fmt.Println("===========================================")
	for _, pcv := range pcvArray {
		fmt.Println(pcv)
	}

}
