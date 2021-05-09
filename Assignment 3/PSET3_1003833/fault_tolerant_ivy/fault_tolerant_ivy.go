package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

const NUM_PROCESSORS int = 10
const NUM_REPLICA int = 3
const NUM_PAGES int = 20
const MAX_SEND_DURATION int = 300 // milliseconds
const MSG_TIMEOUT int = 2         //seconds
const ELECTION_TIMEOUT int = 500  //milliseconds

type CentralManager struct {
	cid             int
	processors      map[int]*Processor
	waitGroup       *sync.WaitGroup
	pageOwner       map[int]int
	pageCopyset     map[int][]int
	otherReplica    map[int]*CentralManager
	election        *Election
	requestChannel  chan Message
	confirmChannel  chan Message
	electionChannel chan ElectionMessage
	killChannel     chan bool
}

type Processor struct {
	pid             int
	cm              *CentralManager
	allReplica      map[int]*CentralManager
	otherProcessors map[int]*Processor
	waitGroup       *sync.WaitGroup
	pageAccess      map[int]AccessType
	pageContent     map[int]string
	contentToWrite  string
	generalChannel  chan Message
	responseChannel chan Message
}

type Election struct {
	coordinatorCid int
	status         ElectionStatus
	rejectChannel  chan ElectionMessage
}

type ElectionStatus int

const (
	RUNNING ElectionStatus = iota
	LOSTELECTION
	NOTINPROGRESS
)

// For easier printing of ElectionStatus
func (e ElectionStatus) String() string {
	return [...]string{
		"RUNNING",
		"LOSTELECTION",
		"NOTINPROGRESS",
	}[e]
}

type Message struct {
	messageType MessageType
	senderID    int
	requesterID int
	page        int
	content     string // if READPAGECONTENT or WRITEPAGECONTENT
}

type MessageType int

const (
	// From processor to CM
	READREQUEST MessageType = iota
	WRITEREQUEST
	READCONFIRM
	WRITECONFIRM
	INVALIDATECONFIRM
	// From CM to processor
	READFORWARD
	WRITEFORWARD
	INVALIDATE
	READNOOWNER
	WRITENOOWNER
	// From processor to processor.
	// Although not in message struct, imagine page contents are sent along with message
	READPAGECONTENT
	WRITEPAGECONTENT
)

// For easier printing of MessageType
func (m MessageType) String() string {
	return [...]string{
		"READREQUEST",
		"WRITEREQUEST",
		"READCONFIRM",
		"WRITECONFIRM",
		"INVALIDATECONFIRM",
		"READFORWARD",
		"WRITEFORWARD",
		"INVALDIATE",
		"READNOOWNER",
		"WRITENOOWNER",
		"READPAGECONTENT",
		"WRITEPAGECONTENT",
	}[m]
}

type AccessType int

const (
	READONLY AccessType = iota
	READWRITE
)

// For easier printing of Access
func (m AccessType) String() string {
	return [...]string{
		"READONLY",
		"READWRITE",
	}[m]
}

type ElectionMessage struct {
	electionMessageType ElectionMessageType
	senderID            int
}

type ElectionMessageType int

const (
	STARTELECTION ElectionMessageType = iota
	REQUEST
	REJECT
	ANNOUNCEWINNER
)

// For easier printing of ElectionMessageType
func (m ElectionMessageType) String() string {
	return [...]string{
		"STARTELECTION",
		"REQUEST",
		"REJECT",
		"ANNOUNCEWINNER",
	}[m]
}

func isInArray(toFind int, array []int) bool {
	for _, element := range array {
		if element == toFind {
			return true
		}
	}
	return false
}

func (cm *CentralManager) sendMessage(msg Message, recipientID int) {
	fmt.Printf("[CM %d] Sending %s message to Processor %d... \n", cm.cid, msg.messageType, recipientID)
	sendDuration := rand.Intn(MAX_SEND_DURATION)
	time.Sleep(time.Millisecond * time.Duration(sendDuration))
	recipientNode := cm.processors[recipientID]
	if msg.messageType == READNOOWNER || msg.messageType == WRITENOOWNER {
		recipientNode.responseChannel <- msg
	} else {
		// READFORWARD, WRITEFORWARD, INVALIDATE
		recipientNode.generalChannel <- msg
	}
}

func (cm *CentralManager) handleReadRequest(requestMsg Message) {
	page := requestMsg.page
	requesterID := requestMsg.requesterID
	// If no records, inform read requester that there is no content. Wait for confirmation to end operation
	if _, exists := cm.pageOwner[page]; !exists {
		noOwnerMsg := Message{
			messageType: READNOOWNER,
			senderID:    0,
			requesterID: requesterID,
			page:        page,
			content:     "",
		}
		go cm.sendMessage(noOwnerMsg, requesterID)
		confirmMsg := <-cm.confirmChannel
		fmt.Printf("[CM %d] Received %s message from Processor %d \n", cm.cid, confirmMsg.messageType, confirmMsg.senderID)
		cm.waitGroup.Done()
		return
	}
	// Send read forward request
	currentOwner := cm.pageOwner[page]
	copyset := cm.pageCopyset[page]
	forwardMsg := Message{
		messageType: READFORWARD,
		senderID:    0,
		requesterID: requesterID,
		page:        page,
		content:     "",
	}
	go cm.sendMessage(forwardMsg, currentOwner)
	// Write to copy set for ownself and ALL OTHER REPLICA
	if !isInArray(requesterID, copyset) {
		copyset = append(copyset, requesterID)
	}
	cm.pageCopyset[page] = copyset
	for _, replica := range cm.otherReplica {
		replica.pageCopyset[page] = copyset
	}
	// Wait for confirmation to complete operation
	confirmMsg := <-cm.confirmChannel
	fmt.Printf("[CM %d] Received %s message from Processor %d \n", cm.cid, confirmMsg.messageType, confirmMsg.senderID)
	cm.waitGroup.Done()
}

func (cm *CentralManager) handleWriteRequest(requestMsg Message) {
	page := requestMsg.page
	requesterID := requestMsg.requesterID
	// If no records, update pageOwner (ALL REPLICA) and inform write requester. Wait for confirm to end operation
	if _, exists := cm.pageOwner[page]; !exists {
		cm.pageOwner[page] = requesterID
		for _, replica := range cm.otherReplica {
			replica.pageOwner[page] = requesterID
		}
		noOwnerMsg := Message{
			messageType: WRITENOOWNER,
			senderID:    0,
			requesterID: requesterID,
			page:        page,
			content:     "",
		}
		go cm.sendMessage(noOwnerMsg, requesterID)
		confirmMsg := <-cm.confirmChannel
		fmt.Printf("[CM %d] Received %s message from Processor %d \n", cm.cid, confirmMsg.messageType, confirmMsg.senderID)
		cm.waitGroup.Done()
		return
	}
	// Invalidate all cached pages and wait for confirmations
	currentOwner := cm.pageOwner[page]
	copyset := cm.pageCopyset[page]
	invalidateMsg := Message{
		messageType: INVALIDATE,
		senderID:    0,
		requesterID: requesterID,
		page:        page,
		content:     "",
	}
	numToInvalidate := len(copyset)
	for _, pid := range copyset {
		go cm.sendMessage(invalidateMsg, pid)
	}
	for i := 0; i < numToInvalidate; i++ {
		<-cm.confirmChannel
	}
	// Send write forward request
	forwardMsg := Message{
		messageType: WRITEFORWARD,
		senderID:    0,
		requesterID: requesterID,
		page:        page,
		content:     "",
	}
	go cm.sendMessage(forwardMsg, currentOwner)
	//  Update pageOwner and clear copyset (ALL REPLICA)
	cm.pageOwner[page] = requesterID
	cm.pageCopyset[page] = []int{}
	for _, replica := range cm.otherReplica {
		replica.pageOwner[page] = requesterID
		replica.pageCopyset[page] = []int{}
	}
	// Wait for confirmation to complete operation.
	confirmMsg := <-cm.confirmChannel
	fmt.Printf("[CM %d] Received %s message from Processor %d \n", cm.cid, confirmMsg.messageType, confirmMsg.senderID)
	cm.waitGroup.Done()
}

func (cm *CentralManager) listenToProcessors() {
	for {
		select {
		case <-cm.killChannel:
			fmt.Printf("[CM %d] Killing processor listener... \n", cm.cid)
			return
		case requestMsg := <-cm.requestChannel:
			fmt.Printf("[CM %d] Received %s message from Processor %d \n", cm.cid, requestMsg.messageType, requestMsg.senderID)
			switch requestMsg.messageType {
			case READREQUEST:
				cm.handleReadRequest(requestMsg)
			case WRITEREQUEST:
				cm.handleWriteRequest(requestMsg)
			}
		}
	}
}

func (cm *CentralManager) sendElectionMessage(recipientReplica *CentralManager, msg ElectionMessage) {
	fmt.Printf("[Replica %d] Sending %s msg to Replica %d \n", cm.cid, msg.electionMessageType, recipientReplica.cid)
	select {
	case recipientReplica.electionChannel <- msg:
		// do nothing because electionMessage is already sent
		fmt.Printf("[Replica %d] Sent %s msg to Replica %d \n", cm.cid, msg.electionMessageType, recipientReplica.cid)
	case <-time.After(time.Millisecond * 200):
		// channel is blocked, means node is dead
		// timeout so do nothing and continue
		fmt.Printf("[Replica %d] Timeout so because Replica %d is probably dead. Skipping... \n", cm.cid, recipientReplica.cid)
	}
}

func (cm *CentralManager) startElection() {
	fmt.Printf("[Replica %d] Replica runs for election \n", cm.cid)
	cm.election.status = RUNNING
	// Clear rejectChannel in case there are rejectMessages from previous elections
	for len(cm.election.rejectChannel) > 0 {
		<-cm.election.rejectChannel
	}
	// Send requestMessage to all replica with higherCids
	num_requests := 0
	for cid, replica := range cm.otherReplica {
		if cid > cm.cid {
			requestMessage := ElectionMessage{
				electionMessageType: REQUEST,
				senderID:            cm.cid,
			}
			go cm.sendElectionMessage(replica, requestMessage)
			num_requests += 1
		}
	}
	// If no replica have higher cid, it automatically wins the election
	// Sleep to let other elections finish first
	if num_requests == 0 {
		time.Sleep(time.Millisecond * time.Duration(ELECTION_TIMEOUT))
		fmt.Printf("[Replica %d] No replica with higher cid so won the election! \n", cm.cid)
		cm.wonElection()
		return
	}
	// Wait for rejection. If timeout, win election
	select {
	case rejectMessage := <-cm.election.rejectChannel:
		fmt.Printf("[Replica %d] Lost election to Replica %d \n", cm.cid, rejectMessage.senderID)
		cm.election.status = LOSTELECTION
	case <-time.After(time.Millisecond * time.Duration(ELECTION_TIMEOUT)):
		fmt.Printf("[Replica %d] Election timeout so won the election! \n", cm.cid)
		cm.wonElection()
	}
}

func (cm *CentralManager) wonElection() {
	fmt.Printf("[Replica %d] Won the election. Broadcasting... \n", cm.cid)
	cm.election.status = NOTINPROGRESS
	cm.election.coordinatorCid = cm.cid
	// Broadcast win message
	for _, replica := range cm.otherReplica {
		broadcastMessage := ElectionMessage{
			electionMessageType: ANNOUNCEWINNER,
			senderID:            cm.cid,
		}
		go cm.sendElectionMessage(replica, broadcastMessage)
	}
	// Start listening to allProcessors
	go cm.listenToProcessors()
	// Give elected CM pointer to allProcessors
	for _, processor := range cm.processors {
		processor.cm = cm
	}

}

func (cm *CentralManager) handleElectionMessage(msg ElectionMessage) {
	fmt.Printf("[Replica %d] Received ElectionMessage %s from Replica %d \n", cm.cid, msg.electionMessageType, msg.senderID)
	switch msg.electionMessageType {
	case STARTELECTION:
		// Previous CM is dead so start a new election
		go cm.startElection()
	case REQUEST:
		// Reply with rejection message
		rejectMessage := ElectionMessage{
			electionMessageType: REJECT,
			senderID:            cm.cid,
		}
		fmt.Printf("[Replica %d] Sending %s msg to Replica %d \n", cm.cid, rejectMessage.electionMessageType, msg.senderID)
		cm.otherReplica[msg.senderID].election.rejectChannel <- rejectMessage
		// Start election if not ongoing
		if cm.election.status == NOTINPROGRESS {
			fmt.Printf("[Replica %d] Starting an election in response to Replica %d \n", cm.cid, msg.senderID)
			go cm.startElection()
		}
	case ANNOUNCEWINNER:
		// End election
		fmt.Printf("[Replica %d] received Replica %d's winning announcement \n", cm.cid, msg.senderID)
		cm.election.coordinatorCid = msg.senderID
		cm.election.status = NOTINPROGRESS
	}

}

func (cm *CentralManager) listenToReplica(allReplica map[int]*CentralManager) {
	fmt.Printf("[Replica %d] Replica is starting... \n", cm.cid)
	// Add pointers to all other replica for easier channel communication
	for _, replica := range allReplica {
		if replica.cid == cm.cid {
			continue
		}
		cm.otherReplica[replica.cid] = replica
	}
	go cm.startElection()
	// Main Loop for receiving ElectionMessages
	for {
		select {
		case msg := <-cm.electionChannel:
			cm.handleElectionMessage(msg)
		case <-cm.killChannel:
			fmt.Printf("[Replica %d] Killing replica listener... \n", cm.cid)
			return
		}
	}
}

func (cm *CentralManager) revive() {
	fmt.Printf("[Replica %d] Replica is reviving... \n", cm.cid)
	// DO NOT START ELECTION so as to not disrupt any ongoing read/write operations
	// Clear channels in case there is leftover from being previous CM
	for len(cm.requestChannel) > 0 {
		<-cm.requestChannel
	}
	for len(cm.confirmChannel) > 0 {
		<-cm.confirmChannel
	}
	for len(cm.electionChannel) > 0 {
		<-cm.electionChannel
	}
	// Main Loop for receiving ElectionMessages
	for {
		select {
		case msg := <-cm.electionChannel:
			cm.handleElectionMessage(msg)
		case <-cm.killChannel:
			fmt.Printf("[Replica %d] Killing replica listener... \n", cm.cid)
			return
		}
	}
}

func (p *Processor) electAndContinue() {
	p.cm = nil
	for _, processors := range p.otherProcessors {
		processors.cm = nil
	}
	// Inform all replica to start election
	for _, replica := range p.allReplica {
		startElectionMsg := ElectionMessage{
			electionMessageType: STARTELECTION,
			senderID:            p.pid,
		}
		select {
		case replica.electionChannel <- startElectionMsg:
			fmt.Printf("[Processor %d] Informed Replica %d to start election \n", p.pid, replica.cid)
		case <-time.After(time.Millisecond * 200):
			fmt.Printf("[Processor %d] Timeout because Replica %d is probably dead. Skipping... \n", p.pid, replica.cid)
		}
	}
	// Busy wait for election to finish and continue with read/write operation
	for p.cm == nil || p.cm.election.status != NOTINPROGRESS {
		// busy wait
		fmt.Printf("[Processor %d] Busy waiting for elected CM to be up... \n", p.pid)
		time.Sleep(time.Second)
	}
	fmt.Printf("[Processor %d] Replica %d is the CM. Continuing read/write operation... \n", p.pid, p.cm.cid)
}

func (p *Processor) sendMessage(msg Message, recipientID int) {
	if recipientID == 0 {
		fmt.Printf("[Processor %d] Sending %s message to Central Manager... \n", p.pid, msg.messageType)
	} else {
		fmt.Printf("[Processor %d] Sending %s message to Processor %d... \n", p.pid, msg.messageType, recipientID)
	}
	sendDuration := rand.Intn(MAX_SEND_DURATION)
	time.Sleep(time.Millisecond * time.Duration(sendDuration))
	if recipientID == 0 {
		if msg.messageType == READREQUEST || msg.messageType == WRITEREQUEST {
			select {
			case p.cm.requestChannel <- msg:
				// Message is sent so do nothing
				fmt.Printf("[Processor %d] Sent %s message to Central Manager... \n", p.pid, msg.messageType)
			case <-time.After(time.Second * time.Duration(MSG_TIMEOUT)):
				// Timeout so CM is probably dead. Start election, wait for election to end, and continue read/write function
				fmt.Printf("[Processor %d] Timeout because CM %d is probably dead. Informing all replica to start election... \n", p.pid, p.cm.cid)
				p.electAndContinue()
				p.sendMessage(msg, 0)
			}
		} else if msg.messageType == READCONFIRM || msg.messageType == WRITECONFIRM || msg.messageType == INVALIDATECONFIRM {
			select {
			case p.cm.confirmChannel <- msg:
				// Message is sent so do nothing
				fmt.Printf("[Processor %d] Sent %s message to Central Manager... \n", p.pid, msg.messageType)
			case <-time.After(time.Second * time.Duration(MSG_TIMEOUT)):
				// Timeout so CM is probably dead. Start election, wait for election to end, and continue read/write function
				fmt.Printf("[Processor %d] Timeout because CM %d is probably dead. Informing all replica to start election... \n", p.pid, p.cm.cid)
				p.electAndContinue()
				p.sendMessage(msg, 0)
			}
		}
	} else {
		// MessageType is READPAGECONTENT or WRITEPAGECONTENT
		p.otherProcessors[recipientID].responseChannel <- msg
	}
}

func (p *Processor) handleReadForward(forwardMsg Message) {
	page := forwardMsg.page
	requesterID := forwardMsg.requesterID
	// Send the page to processor that requested to read
	contentMsg := Message{
		messageType: READPAGECONTENT,
		senderID:    p.pid,
		requesterID: requesterID,
		page:        page,
		content:     p.pageContent[page],
	}
	go p.sendMessage(contentMsg, requesterID)
}

func (p *Processor) handleWriteForward(forwardMsg Message) {
	page := forwardMsg.page
	requesterID := forwardMsg.requesterID
	// Send page to processor that requested to write
	contentMsg := Message{
		messageType: WRITEPAGECONTENT,
		senderID:    p.pid,
		requesterID: requesterID,
		page:        page,
		content:     p.pageContent[page],
	}
	go p.sendMessage(contentMsg, requesterID)
	// Invalidate own copy
	delete(p.pageAccess, page)
}

func (p *Processor) handleInvalidate(invalidateMsg Message) {
	page := invalidateMsg.page
	// Invalidate cached copy
	delete(p.pageAccess, page)
	// Send invalidate confirmation to Central Manager
	confirmMsg := Message{
		messageType: INVALIDATECONFIRM,
		senderID:    p.pid,
		requesterID: invalidateMsg.requesterID,
		page:        page,
		content:     "",
	}
	go p.sendMessage(confirmMsg, 0)
}

func (p *Processor) handleReadNoOwner(noOwnerMsg Message) {
	page := noOwnerMsg.page
	fmt.Printf("[Processor %d] {READ} Page %d had not been written yet. \n", p.pid, page)
	// Send read confirmation to Central Manager
	confirmMsg := Message{
		messageType: READCONFIRM,
		senderID:    p.pid,
		requesterID: noOwnerMsg.requesterID,
		page:        page,
		content:     "",
	}
	go p.sendMessage(confirmMsg, 0)
}

func (p *Processor) handleWriteNoOwner(noOwnerMsg Message) {
	page := noOwnerMsg.page
	fmt.Printf("[Processor %d] {WRITE} Page %d had not been written yet. \n", p.pid, page)
	// Write own content and mark page as read/write page
	p.pageContent[page] = p.contentToWrite
	p.pageAccess[page] = READWRITE
	fmt.Printf("[Processor %d] {WRITE} Successfully wrote Page %d content {%s} \n", p.pid, page, p.contentToWrite)
	// Send write confirmation to Central Manager
	confirmMsg := Message{
		messageType: WRITECONFIRM,
		senderID:    p.pid,
		requesterID: noOwnerMsg.requesterID,
		page:        page,
		content:     "",
	}
	go p.sendMessage(confirmMsg, 0)
}

func (p *Processor) handleReadPageContent(contentMsg Message) {
	page := contentMsg.page
	content := contentMsg.content
	// Cache content and mark page as read only page
	p.pageContent[page] = content
	p.pageAccess[page] = READONLY
	fmt.Printf("[Processor %d] {READ} Page %d received from owner. Contents of page: {%s} \n", p.pid, page, content)
	// Send read confirmation to Central Manager
	confirmMsg := Message{
		messageType: READCONFIRM,
		senderID:    p.pid,
		requesterID: contentMsg.requesterID,
		page:        page,
		content:     "",
	}
	go p.sendMessage(confirmMsg, 0)
}

func (p *Processor) handleWritePageContent(contentMsg Message) {
	page := contentMsg.page
	fmt.Printf("[Processor %d] {WRITE} Received Processor %d's old Page %d content {%s} \n", p.pid, contentMsg.senderID, page, contentMsg.content)
	// Write own content and mark page as read/write page
	p.pageContent[page] = p.contentToWrite
	p.pageAccess[page] = READWRITE
	fmt.Printf("[Processor %d] {WRITE} Successfully wrote Page %d content {%s} \n", p.pid, page, p.contentToWrite)
	// Send write confirmation to Central Manager
	confirmMsg := Message{
		messageType: WRITECONFIRM,
		senderID:    p.pid,
		requesterID: contentMsg.requesterID,
		page:        page,
		content:     "",
	}
	go p.sendMessage(confirmMsg, 0)
}

func (p *Processor) listen() {
	for {
		msg := <-p.generalChannel
		fmt.Printf("[Processor %d] Received %s message from Central Manager \n", p.pid, msg.messageType)
		switch msg.messageType {
		case READFORWARD:
			p.handleReadForward(msg)
		case WRITEFORWARD:
			p.handleWriteForward(msg)
		case INVALIDATE:
			p.handleInvalidate(msg)
		}
	}
}

func (p *Processor) read(page int) {
	p.waitGroup.Add(1)
	// Wait for CM to be elected
	for p.cm == nil || p.cm.election.status != NOTINPROGRESS {
		// busy wait
		fmt.Printf("[Processor %d] Busy waiting for elected CM to be up... \n", p.pid)
		time.Sleep(time.Second)
	}
	fmt.Printf("[Processor %d] Replica %d is the CM. Continuing read/write operation... \n", p.pid, p.cm.cid)
	// If no page fault, read cached content
	if _, exists := p.pageAccess[page]; exists {
		println("exists", p.pageAccess)
		content := p.pageContent[page]
		fmt.Printf("[Processor %d] {READ} Page %d cached. Contents of page: {%s} \n", p.pid, page, content)
		p.waitGroup.Done()
		return
	}
	// Page fault, so request read from CM. Content read will be printed when page content is sent over from page owner
	requestMsg := Message{
		messageType: READREQUEST,
		senderID:    p.pid,
		requesterID: p.pid,
		page:        page,
		content:     "",
	}
	go p.sendMessage(requestMsg, 0)
	// Handle accordingly
	select {
	case msg := <-p.responseChannel:
		switch msg.messageType {
		case READNOOWNER:
			fmt.Printf("[Processor %d] Received %s message from Central Manager \n", p.pid, msg.messageType)
			p.handleReadNoOwner(msg)
		case READPAGECONTENT:
			fmt.Printf("[Processor %d] Received %s message from Processor %d \n", p.pid, msg.messageType, msg.senderID)
			p.handleReadPageContent(msg)
		}
	case <-time.After(time.Second * time.Duration(MSG_TIMEOUT*2)):
		// Timeout scenario (sent to CM requestChannel but CM crashes before sending READFORWARD/READNOOWNER). Restart read request
		// Note: Timeout here has to be longer than timeout at sendmessage, else both will happen
		p.read(page)
	}
}

func (p *Processor) write(page int, content string) {
	p.waitGroup.Add(1)
	// Wait for CM to be elected
	for p.cm == nil {
		// busy wait
		fmt.Printf("[Processor %d] Busy waiting for elected CM to be up... \n", p.pid)
		time.Sleep(time.Second)
	}
	fmt.Printf("[Processor %d] Replica %d is the CM. Continuing read/write operation... \n", p.pid, p.cm.cid)
	// If no page fault, do nothing
	if accessType, exists := p.pageAccess[page]; exists {
		if accessType == READWRITE && p.pageContent[page] == content {
			fmt.Printf("[Processor %d] {WRITE} Page %d already has content {%s} \n", p.pid, page, content)
			p.waitGroup.Done()
			return
		}
	}
	// Page fault, so request write from CM
	p.contentToWrite = content
	requestMsg := Message{
		messageType: WRITEREQUEST,
		senderID:    p.pid,
		requesterID: p.pid,
		page:        page,
		content:     content,
	}
	go p.sendMessage(requestMsg, 0)
	// Handle accordingly
	select {
	case msg := <-p.responseChannel:
		switch msg.messageType {
		case WRITENOOWNER:
			fmt.Printf("[Processor %d] Received %s message from Central Manager \n", p.pid, msg.messageType)
			p.handleWriteNoOwner(msg)
		case WRITEPAGECONTENT:
			fmt.Printf("[Processor %d] Received %s message from Processor %d \n", p.pid, msg.messageType, msg.senderID)
			p.handleWritePageContent(msg)
		}
	case <-time.After(time.Second * time.Duration(MSG_TIMEOUT*2)):
		// Timeout scenario (sent to CM requestChannel but CM crashes before sending WRITEFORWARD/WRITENOOWNER). Restart write request
		// Note: Timeout here has to be longer than timeout at sendmessage, else both will happen
		p.write(page, content)
	}
}

func main() {

	// Wait group to wait later
	var wg sync.WaitGroup

	// Create processors
	allProcessors := make(map[int]*Processor)
	for i := 1; i <= NUM_PROCESSORS; i++ {
		p := &Processor{
			pid:             i,
			cm:              nil,
			allReplica:      nil,
			otherProcessors: make(map[int]*Processor),
			waitGroup:       &wg,
			pageAccess:      make(map[int]AccessType),
			pageContent:     make(map[int]string),
			contentToWrite:  "",
			generalChannel:  make(chan Message),
			responseChannel: make(chan Message),
		}
		allProcessors[p.pid] = p
	}
	// Give every processor pointers to every other processor
	for _, p1 := range allProcessors {
		for _, p2 := range allProcessors {
			if p1.pid != p2.pid {
				p1.otherProcessors[p2.pid] = p2
			}
		}
	}

	// Create central managers
	allReplica := make(map[int]*CentralManager)
	for i := 1; i <= NUM_REPLICA; i++ {
		cm := &CentralManager{
			cid:          i,
			processors:   allProcessors,
			waitGroup:    &wg,
			pageOwner:    make(map[int]int),
			pageCopyset:  make(map[int][]int),
			otherReplica: make(map[int]*CentralManager),
			election: &Election{
				coordinatorCid: 0,
				status:         NOTINPROGRESS,
				rejectChannel:  make(chan ElectionMessage, NUM_REPLICA),
			},
			requestChannel:  make(chan Message),
			confirmChannel:  make(chan Message),
			electionChannel: make(chan ElectionMessage),
			killChannel:     make(chan bool),
		}
		allReplica[cm.cid] = cm
	}
	// Give every processor pointers to allReplica
	for i := 1; i <= NUM_PROCESSORS; i++ {
		allProcessors[i].allReplica = allReplica
	}

	// Start first election and listen for election calls
	for i := 1; i <= NUM_REPLICA; i++ {
		go allReplica[i].listenToReplica(allReplica)
	}

	// All processors listen for messages from Central Manager and other processors
	for _, p := range allProcessors {
		go p.listen()
	}

	// TEST TEST
	/* allProcessors[2].read(3)
	allProcessors[1].write(3, "This is written by pid 1")
	allProcessors[3].read(3)
	allProcessors[2].read(3)
	allProcessors[2].write(3, "This is written by pid 2")
	allProcessors[1].read(3)
	allProcessors[3].read(3)
	wg.Wait() */

	// For EXPERIMENT 1: Run the following functions and record the timing.
	start := time.Now()
	for i := 1; i <= NUM_PROCESSORS; i++ {
		allProcessors[i].read(i)
	}
	for i := 1; i <= NUM_PROCESSORS; i++ {
		toWrite := fmt.Sprintf("This is written by pid %d", i)
		allProcessors[i].write(i, toWrite)
	}
	for i := 1; i <= NUM_PROCESSORS; i++ {
		temp := i + 1
		temp %= (NUM_PAGES + 1)
		if temp == 0 {
			temp += 1
		}
		allProcessors[i].read(temp)
	}
	for i := 1; i <= NUM_PROCESSORS; i++ {
		toWrite := fmt.Sprintf("This is written by pid %d", i)
		temp := i + 1
		temp %= (NUM_PAGES + 1)
		if temp == 0 {
			temp += 1
		}
		allProcessors[i].write(temp, toWrite)
	}
	wg.Wait()
	end := time.Now()
	time.Sleep(time.Second * 3)
	fmt.Printf("Time taken = %.2f seconds \n", end.Sub(start).Seconds())

	// For EXPERIMENT 2: (a) Kill the primary CM and (b) make it restart
	/* allProcessors[2].read(3)
	allProcessors[1].write(3, "This is written by pid 1")
	allProcessors[3].read(3)
	// Simulate primary CM fail. 2 msgs bcos need to stop both processor and replica (election) listener
	allReplica[3].killChannel <- true
	allReplica[3].killChannel <- true
	allProcessors[2].read(3)
	allProcessors[2].write(3, "This is written by pid 2")
	allProcessors[1].read(3)
	// Simulate primary CM restart. Comment out this part for Scenario (a)
	go allReplica[3].revive()
	allProcessors[3].read(3)
	wg.Wait() */

}
