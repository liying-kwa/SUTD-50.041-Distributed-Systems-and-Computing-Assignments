package main

import (
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"sync"
	"time"
)

const NUM_NODES int = 10
const CS_DURATION int = 1         // seconds
const MAX_SEND_DURATION int = 300 // milliseconds

type Node struct {
	nodeID         int
	messageChannel chan Message
	nodeMap        map[int]*Node // All other machines
	queue          []Message

	// replyRecords are to keep track whether other nodes have replied to requests sent from this node
	// Only need a replyRecord for requests that originate from this node
	// replyRecords e.g. { req(T1): {nodeID: true, nodeID: false, ...}, req(T2): {...} }
	// One record is for one req(T). e.g. {nodeID: true, nodeID: false, ...}
	replyRecords map[int]map[int]bool

	// repliesOnHold is for replies to REQUEST messages that are on hold
	// Replies are added here if this node receives a reply from a node that sent a request, and needs to hold onto a reply to that node
	// Hold on to this reply if THAT node has not replied to a earlier request from THIS node
	// e.g. (THAT) nodeID: {logicalTS of req(T) from THAT node : req(T) from THAT node}
	repliesOnHold map[int]map[int]Message

	waitGroup    *sync.WaitGroup
	logicalClock int
}

type Message struct {
	messageType MessageType
	senderID    int
	logicalTS   int

	// This is to keep track of where req(T) originated from and its request timestamp T
	requesterID int
	requestTS   int
}

type MessageType int

const (
	REQUEST MessageType = iota
	REPLY
	RELEASE
)

// For easier printing of MessageType
func (m MessageType) String() string {
	return [...]string{"REQUEST", "REPLY", "RELEASE"}[m]
}

func queueString(queue []Message) string {
	if len(queue) == 0 {
		return "empty"
	}
	toReturn := "|"
	for _, msg := range queue {
		toReturn += " "
		toReturn += strconv.Itoa(msg.requesterID) + "." + strconv.Itoa(msg.requestTS)
		toReturn += " |"
	}
	return toReturn
}

func (n *Node) addToQueue(msg Message) {
	n.queue = append(n.queue, msg)
	// Sort queue based on requestTS, then senderID
	sort.SliceStable(n.queue, func(i, j int) bool {
		if n.queue[i].requestTS < n.queue[j].requestTS {
			return true
		} else if n.queue[i].requestTS == n.queue[j].requestTS && n.queue[i].senderID < n.queue[j].senderID {
			return true
		} else {
			return false
		}
	})
	fmt.Printf("[Node %d] Request %d.%d added to queue. Queue: [ %s ] \n", n.nodeID, msg.requesterID, msg.requestTS, queueString(n.queue))
}

func (n *Node) removeFromQueue() {
	removedMsg := n.queue[0]
	n.queue = n.queue[1:]
	fmt.Printf("[Node %d] Request %d.%d removed from queue. Queue: [ %s ] \n", n.nodeID, removedMsg.requesterID, removedMsg.requestTS, queueString(n.queue))
}

func (n *Node) sendMessage(msg Message, recipientID int) {
	fmt.Printf("[Node %d] Sending %s message %d.%d to Node %d... \n", n.nodeID, msg.messageType, msg.requesterID, msg.requestTS, recipientID)
	sendDuration := rand.Intn(MAX_SEND_DURATION)
	time.Sleep(time.Millisecond * time.Duration(sendDuration))
	recipientNode := n.nodeMap[recipientID]
	recipientNode.messageChannel <- msg
}

func (n *Node) requestCS() {
	n.logicalClock += 1
	logicalTS := n.logicalClock
	fmt.Printf("[Node %d] Node is requesting to enter CS. Request message is %d.%d \n", n.nodeID, n.nodeID, logicalTS)
	requestMsg := Message{
		messageType: REQUEST,
		senderID:    n.nodeID,
		logicalTS:   logicalTS,
		requesterID: n.nodeID,
		requestTS:   logicalTS,
	}
	// Add request to Qi (and update replyRecords). If repliesOnHold entries for other nodes are not created yet, create.
	n.addToQueue(requestMsg)
	replyRecord := make(map[int]bool)
	for nodeID := range n.nodeMap {
		replyRecord[nodeID] = false
	}
	n.replyRecords[requestMsg.logicalTS] = replyRecord
	for nodeID := range n.nodeMap {
		if _, exist := n.repliesOnHold[nodeID]; !exist {
			n.repliesOnHold[nodeID] = make(map[int]Message)
		}
	}
	// If this is the only node, SKIP EVERYTHING AND EXECUTE CS
	if len(n.nodeMap) == 0 {
		go n.executeCS()
		return
	}
	// Broadcast req(T) to all other machines
	for nodeID := range n.nodeMap {
		go n.sendMessage(requestMsg, nodeID)
	}
}

func (n *Node) executeCS() {
	n.logicalClock += 1
	logicalTS := n.logicalClock
	fmt.Printf("[Node %d] Executing CS with request %d.%d... \n", n.nodeID, n.queue[0].requesterID, n.queue[0].requestTS)
	time.Sleep(time.Second * time.Duration(CS_DURATION))
	releaseMsg := Message{
		messageType: RELEASE,
		senderID:    n.nodeID,
		logicalTS:   logicalTS,
		requesterID: n.queue[0].requesterID,
		requestTS:   n.queue[0].requestTS,
	}
	// Pop head of Qi
	n.removeFromQueue()
	// Send RELEASE message to all machines
	for nodeID := range n.nodeMap {
		go n.sendMessage(releaseMsg, nodeID)
	}
	fmt.Printf("[Node %d] Exiting CS with request %d.%d... \n", n.nodeID, releaseMsg.requesterID, releaseMsg.requestTS)
	n.waitGroup.Done()
}

func (n *Node) replyOrHold(requestMsg Message) {
	pending := false
	for requestTS, replyRecord := range n.replyRecords {
		if requestTS < requestMsg.requestTS {
			// Earlier msg
			if replyRecord[requestMsg.senderID] == false {
				pending = true
			}
		} else if requestTS == requestMsg.requestTS && n.nodeID < requestMsg.senderID {
			// If msgs have same logical TS, lower ID node have priority, so take it to be an earlier msg
			if replyRecord[requestMsg.senderID] == false {
				pending = true
			}
		}
	}
	if pending == false {
		// Reply to req(T)
		n.logicalClock += 1
		logicalTS := n.logicalClock
		replyMsg := Message{
			messageType: REPLY,
			senderID:    n.nodeID,
			logicalTS:   logicalTS,
			requesterID: requestMsg.requesterID,
			requestTS:   requestMsg.requestTS,
		}
		go n.sendMessage(replyMsg, requestMsg.senderID)
		// Remove from repliesOnHold, if present
		delete(n.repliesOnHold[requestMsg.senderID], requestMsg.logicalTS)
	} else {
		// Hold reply to req(T)
		n.repliesOnHold[requestMsg.senderID][requestMsg.logicalTS] = requestMsg
		fmt.Printf("[Node %d] Holding on to reply %d.%d to Node %d \n", n.nodeID, requestMsg.requesterID, requestMsg.requestTS, requestMsg.senderID)
	}
}

func (n *Node) onReceiveRequest(requestMsg Message) {
	// Add req(T) to Qi
	n.addToQueue(requestMsg)
	// Check whether any reply is pending for an earlier request req(T’) in Qi.
	// If pending, hold reply to Qi. Otherwise, reply to req(T)
	n.replyOrHold(requestMsg)
}

func (n *Node) onReceiveReply(replyMsg Message) {
	/* // If req(T) does not exist in replyRecords yet, create an entry for it. Should not be facing this scenario though...
	ts := msg.logicalTS
	if _, exists := n.replyRecords[ts]; !exists {
		record := make(map[int]bool)
		for nodeID := range n.nodeMap {
			record[nodeID] = false
		}
		n.replyRecords[ts] = record
	} */
	n.replyRecords[replyMsg.requestTS][replyMsg.senderID] = true
	// Check if there are messages (to the node that replied) that are on hold
	for _, requestMsg := range n.repliesOnHold[replyMsg.senderID] {
		fmt.Printf("[Node %d] Re-checking REQUEST message %d.%d \n", n.nodeID, requestMsg.requesterID, requestMsg.requestTS)
		n.replyOrHold(requestMsg)
	}
	// Check if everyone has replied this Req from this Node and it is at the front of the queue.
	allReplied := true
	for _, nodeHasReplied := range n.replyRecords[replyMsg.requestTS] {
		if nodeHasReplied == false {
			allReplied = false
			break
		}
	}
	if allReplied == true {
		fmt.Printf("[Node %d] All replies have been received for Request %d.%d \n", n.nodeID, replyMsg.requesterID, replyMsg.requestTS)
		firstRequest := n.queue[0]
		if firstRequest.senderID == n.nodeID && firstRequest.requestTS == replyMsg.requestTS {
			fmt.Printf("[Node %d] Request %d.%d is also at the front of the queue. Executing CS... \n", n.nodeID, replyMsg.requesterID, replyMsg.requestTS)
			delete(n.replyRecords, replyMsg.requestTS)
			n.executeCS()
		}
	}
}

func (n *Node) onReceiveRelease(releaseMsg Message) {
	// Pop head of Qi
	n.removeFromQueue()
	// If first message in Qi (is from this node and) has received all replies, let it enter CS
	if len(n.queue) >= 1 {
		firstRequest := n.queue[0]
		if firstRequest.senderID == n.nodeID {
			allReplied := true
			for _, nodeHasReplied := range n.replyRecords[firstRequest.requestTS] {
				if nodeHasReplied == false {
					allReplied = false
					break
				}
			}
			if allReplied == true {
				fmt.Printf("[Node %d] First request %d.%d has all replies. Executing CS... \n", n.nodeID, firstRequest.requesterID, firstRequest.requestTS)
				delete(n.replyRecords, firstRequest.requestTS)
				n.executeCS()
			}
		}
	}
}

func (n *Node) listen() {
	for {
		msg := <-n.messageChannel
		fmt.Printf("[Node %d] Received %s message %d.%d from Node %d \n", n.nodeID, msg.messageType, msg.requesterID, msg.requestTS, msg.senderID)
		if msg.logicalTS >= n.logicalClock {
			n.logicalClock = msg.logicalTS + 1
		} else {
			n.logicalClock += 1
		}

		if msg.messageType == REQUEST {
			n.onReceiveRequest(msg)
		} else if msg.messageType == REPLY {
			n.onReceiveReply(msg)
		} else if msg.messageType == RELEASE {
			n.onReceiveRelease(msg)
		}
	}
}

func main() {

	// Wait group to wait later
	var wg sync.WaitGroup

	// Create nodes
	globalNodeMap := make(map[int]*Node)
	for i := 1; i <= NUM_NODES; i++ {
		n := &Node{
			nodeID:         i,
			messageChannel: make(chan Message),
			nodeMap:        make(map[int]*Node),
			queue:          []Message{},
			replyRecords:   make(map[int]map[int]bool),
			repliesOnHold:  make(map[int]map[int]Message),
			waitGroup:      &wg,
			logicalClock:   0,
		}
		globalNodeMap[n.nodeID] = n
	}

	// Give every node pointers to every other node
	for _, n1 := range globalNodeMap {
		for _, n2 := range globalNodeMap {
			if n1.nodeID != n2.nodeID {
				n1.nodeMap[n2.nodeID] = n2
			}
		}
	}

	// All nodes need to listen for requests from other nodes
	for _, n := range globalNodeMap {
		go n.listen()
	}

	// Ask to enter critical section
	start := time.Now()
	for _, node := range globalNodeMap {
		go node.requestCS()
		wg.Add(1)
	}
	wg.Wait()

	// This is a test to see if the logicalTS works. Uncomment to see
	// (Have each node send a second req to execute CS. But make sure it doesnt send this before the previous req has exited the CS)
	/* for _, node := range globalNodeMap {
		go node.requestCS()
		wg.Add(1)
	}
	wg.Wait() */

	// Wait for last node to exit CS and record the time taken
	end := time.Now()
	time.Sleep(time.Second * 3)
	fmt.Printf("Time taken = %.2f seconds \n", end.Sub(start).Seconds())

}
