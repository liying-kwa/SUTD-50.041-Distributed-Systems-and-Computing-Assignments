package main

import (
	"fmt"
	"time"
)

const NUM_NODES int = 5
const ELECTION_TIMEOUT int = 2

type Node struct {
	pid            int
	otherNodes     map[int]*Node
	election       *Election
	messageChannel chan Message
	killChannel    chan bool
}

type Election struct {
	coordinatorPid int
	status         ElectionStatus
	rejectChannel  chan Message
}

type ElectionStatus int

const (
	Running ElectionStatus = iota
	LostElection
	NotInProgress
)

// For easier printing of ElectionStatus
func (e ElectionStatus) String() string {
	return [...]string{"Running", "LostElection", "NotInProgress"}[e]
}

type Message struct {
	messageType MessageType
	senderID    int
}

type MessageType int

const (
	Request MessageType = iota
	Reject
	AnnounceWinner
)

// For easier printing of MessageType
func (m MessageType) String() string {
	return [...]string{"Request", "Reject", "AnnounceWinner"}[m]
}

func createNode(pid int) *Node {
	node := &Node{
		pid,
		make(map[int]*Node),
		&Election{
			0,
			NotInProgress,
			make(chan Message, NUM_NODES),
		},
		make(chan Message),
		make(chan bool),
	}
	return node
}

func (n *Node) start(allNodes []*Node) {

	fmt.Printf("Node %d is starting... \n", n.pid)
	// Add pointers to all other machines for easier channel communication
	for _, node := range allNodes {
		if node.pid == n.pid {
			continue
		}
		n.otherNodes[node.pid] = node
	}
	go n.startElection()
	// Main Loop for receiving messages
	for {
		select {
		case msg := <-n.messageChannel:
			n.handleMessage(msg)
		case <-n.killChannel:
			return
		}
	}
}

func (n *Node) handleMessage(msg Message) {
	fmt.Printf("Node %d received message %s from node %d \n", n.pid, msg.messageType, msg.senderID)
	switch msg.messageType {
	case Request:
		// Reply with rejection message
		fmt.Printf("Node %d is sending rejection to Node %d \n", n.pid, msg.senderID)
		rejectMessage := Message{Reject, n.pid}
		n.otherNodes[msg.senderID].election.rejectChannel <- rejectMessage
		// Start election if not ongoing
		if n.election.status == NotInProgress {
			fmt.Printf("Node %d is starting an election in response to Node %d \n", n.pid, msg.senderID)
			go n.startElection()
		}
	case AnnounceWinner:
		// End election
		fmt.Printf("Node %d received Node %d's winning announcement \n", n.pid, msg.senderID)
		n.election.coordinatorPid = msg.senderID
		n.election.status = NotInProgress
	}

}

func (n *Node) startElection() {
	fmt.Printf("Node %d runs for election \n", n.pid)
	n.election.status = Running
	// Clear rejectChannel in case there are rejectMessages from previous elections
	for len(n.election.rejectChannel) > 0 {
		<-n.election.rejectChannel
	}
	// Send requestMessage to all nodes with higherPids
	num_requests := 0
	for pid, node := range n.otherNodes {
		if pid > n.pid {
			fmt.Printf("Node %d sends a request to Node %d \n", n.pid, node.pid)
			requestMessage := Message{Request, n.pid}
			go n.sendMessage(node, requestMessage)
			num_requests += 1
		}
	}
	// If no machines have higher pid, it automatically wins the election
	// Sleep to let other elections finish first
	if num_requests == 0 {
		time.Sleep(time.Second * time.Duration(ELECTION_TIMEOUT))
		n.wonElection()
		return
	}
	// Wait for rejection. If timeout, win election
	select {
	case rejectMessage := <-n.election.rejectChannel:
		fmt.Printf("Node %d has lost election to Node %d \n", n.pid, rejectMessage.senderID)
		n.election.status = LostElection
	case <-time.After(time.Second * time.Duration(ELECTION_TIMEOUT)):
		n.wonElection()
	}
}

func (n *Node) wonElection() {
	fmt.Printf("Node %d has won the election. Broadcasting... \n", n.pid)
	n.election.status = NotInProgress
	n.election.coordinatorPid = n.pid
	// Broadcast win message
	for pid, node := range n.otherNodes {
		fmt.Printf("Node %d sends AnnounceWinner msg to Node %d \n", n.pid, pid)
		broadcastMessage := Message{AnnounceWinner, n.pid}
		go n.sendMessage(node, broadcastMessage)
	}
}

func (n *Node) sendMessage(receivingNode *Node, msg Message) {
	select {
	case receivingNode.messageChannel <- msg:
		// do nothing because message is sent already
	case <-time.After(time.Millisecond * 200):
		// channel is blocked, means node is dead
		// timeout so do nothing and continue
	}
}
