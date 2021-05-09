package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"
)

const NUM_NODES int = 10
const CS_DURATION int = 1         // seconds
const MAX_SEND_DURATION int = 300 // milliseconds

type Server struct {
	messageChannel chan Message
	queue          []Message
	nodeMap        map[int]*Node
}

type Node struct {
	nodeID         int
	messageChannel chan Message
	server         *Server
	waitGroup      *sync.WaitGroup
}

type Message struct {
	messageType MessageType
	senderID    int
}

type MessageType int

const (
	REQUEST MessageType = iota
	OK
	RELEASE
)

// For easier printing of MessageType
func (m MessageType) String() string {
	return [...]string{"REQUEST", "OK", "RELEASE"}[m]
}

func queueString(queue []Message) string {
	if len(queue) == 0 {
		return "empty"
	}
	toReturn := "|"
	for _, msg := range queue {
		toReturn += " "
		toReturn += strconv.Itoa(msg.senderID)
		toReturn += " |"
	}
	return toReturn
}

func (s *Server) addToQueue(msg Message) {
	s.queue = append(s.queue, msg)
	fmt.Printf("[Server] Node %d added to queue. Queue: [ %s ] \n", msg.senderID, queueString(s.queue))
}

func (s *Server) removeFromQueue() {
	removedID := s.queue[0].senderID
	s.queue = s.queue[1:]
	fmt.Printf("[Server] Node %d removed from queue. Queue: [ %s ] \n", removedID, queueString(s.queue))
}

func (s *Server) sendMessage(msg Message, recipientID int) {
	fmt.Printf("[Server] Sending a message to Node %d... \n", recipientID)
	sendDuration := rand.Intn(MAX_SEND_DURATION)
	time.Sleep(time.Millisecond * time.Duration(sendDuration))
	recipientNode := s.nodeMap[recipientID]
	recipientNode.messageChannel <- msg
}

func (s *Server) listen() {
	for {

		msg := <-s.messageChannel
		fmt.Printf("[Server] Received %s message from Node %d \n", msg.messageType, msg.senderID)

		if msg.messageType == REQUEST {
			s.addToQueue(msg)
			if len(s.queue) == 1 {
				nextNodeID := s.queue[0].senderID
				okMsg := Message{OK, 0}
				go s.sendMessage(okMsg, nextNodeID)
			}

		} else if msg.messageType == RELEASE {
			s.removeFromQueue()
			if len(s.queue) >= 1 {
				nextNodeID := s.queue[0].senderID
				okMsg := Message{OK, 0}
				go s.sendMessage(okMsg, nextNodeID)
			}
		}

	}
}

func (n *Node) sendMessage(msg Message) {
	fmt.Printf("[Node %d] Sending %s message to server... \n", n.nodeID, msg.messageType)
	sendDuration := rand.Intn(MAX_SEND_DURATION)
	time.Sleep(time.Millisecond * time.Duration(sendDuration))
	n.server.messageChannel <- msg
}

func (n *Node) requestCS() {
	fmt.Printf("[Node %d] Node is requesting to enter CS. \n", n.nodeID)
	requestMsg := Message{REQUEST, n.nodeID}
	n.sendMessage(requestMsg)
	<-n.messageChannel
	fmt.Printf("[Node %d] Received message OK from server \n", n.nodeID)
	n.executeCS()
}

func (n *Node) executeCS() {
	fmt.Printf("[Node %d] Entering CS... \n", n.nodeID)
	time.Sleep(time.Second * time.Duration(CS_DURATION))
	releaseMsg := Message{RELEASE, n.nodeID}
	n.sendMessage(releaseMsg)
	fmt.Printf("[Node %d] Exiting CS... \n", n.nodeID)
	n.waitGroup.Done()
}

func main() {

	// Wait group to wait later
	var wg sync.WaitGroup

	// Create server
	s := &Server{
		messageChannel: make(chan Message),
		queue:          []Message{},
		nodeMap:        make(map[int]*Node),
	}

	// Create nodes
	for i := 1; i <= NUM_NODES; i++ {
		n := &Node{
			nodeID:         i,
			messageChannel: make(chan Message),
			server:         s,
			waitGroup:      &wg,
		}
		s.nodeMap[n.nodeID] = n
	}

	// Server listens for messages from any node and responds correspondingly
	go s.listen()

	// Ask to enter critical section
	start := time.Now()
	for _, node := range s.nodeMap {
		go node.requestCS()
		wg.Add(1)
	}

	// Wait for last node to exit CS
	wg.Wait()
	end := time.Now()

	// Record time taken
	time.Sleep(time.Second * 3)
	fmt.Printf("Time taken = %.2f seconds \n", end.Sub(start).Seconds())

}
