package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

const NUM_PROCESSORS int = 10
const NUM_PAGES int = 20
const MAX_SEND_DURATION int = 300 // milliseconds

type CentralManager struct {
	processors     map[int]*Processor
	waitGroup      *sync.WaitGroup
	pageOwner      map[int]int
	pageCopyset    map[int][]int
	requestChannel chan Message
	confirmChannel chan Message
}

type Processor struct {
	pid             int
	cm              *CentralManager
	otherProcessors map[int]*Processor
	waitGroup       *sync.WaitGroup
	pageAccess      map[int]AccessType
	pageContent     map[int]string
	contentToWrite  string
	generalChannel  chan Message
	responseChannel chan Message
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

func isInArray(toFind int, array []int) bool {
	for _, element := range array {
		if element == toFind {
			return true
		}
	}
	return false
}

func (cm *CentralManager) sendMessage(msg Message, recipientID int) {
	fmt.Printf("[CM] Sending %s message to Processor %d... \n", msg.messageType, recipientID)
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
		fmt.Printf("[CM] Received %s message from Processor %d \n", confirmMsg.messageType, confirmMsg.senderID)
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
	// Write to copy set
	if !isInArray(requesterID, copyset) {
		copyset = append(copyset, requesterID)
	}
	cm.pageCopyset[page] = copyset
	// Wait for confirmation to complete operation
	confirmMsg := <-cm.confirmChannel
	fmt.Printf("[CM] Received %s message from Processor %d \n", confirmMsg.messageType, confirmMsg.senderID)
	cm.waitGroup.Done()
}

func (cm *CentralManager) handleWriteRequest(requestMsg Message) {
	page := requestMsg.page
	requesterID := requestMsg.requesterID
	// If no records, update pageOwner and inform write requester. Wait for confirm to end operation
	if _, exists := cm.pageOwner[page]; !exists {
		cm.pageOwner[page] = requesterID
		noOwnerMsg := Message{
			messageType: WRITENOOWNER,
			senderID:    0,
			requesterID: requesterID,
			page:        page,
			content:     "",
		}
		go cm.sendMessage(noOwnerMsg, requesterID)
		confirmMsg := <-cm.confirmChannel
		fmt.Printf("[CM] Received %s message from Processor %d \n", confirmMsg.messageType, confirmMsg.senderID)
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
	// Wait for confirmation to complete operation
	confirmMsg := <-cm.confirmChannel
	fmt.Printf("[CM] Received %s message from Processor %d \n", confirmMsg.messageType, confirmMsg.senderID)
	cm.pageOwner[page] = requesterID
	cm.pageCopyset[page] = []int{}
	cm.waitGroup.Done()
}

func (cm *CentralManager) listen() {
	for {
		requestMsg := <-cm.requestChannel
		fmt.Printf("[CM] Received %s message from Processor %d \n", requestMsg.messageType, requestMsg.senderID)
		switch requestMsg.messageType {
		case READREQUEST:
			cm.handleReadRequest(requestMsg)
		case WRITEREQUEST:
			cm.handleWriteRequest(requestMsg)
		}
	}
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
			p.cm.requestChannel <- msg
		} else if msg.messageType == READCONFIRM || msg.messageType == WRITECONFIRM || msg.messageType == INVALIDATECONFIRM {
			p.cm.confirmChannel <- msg
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
	msg := <-p.responseChannel
	switch msg.messageType {
	case READNOOWNER:
		fmt.Printf("[Processor %d] Received %s message from Central Manager \n", p.pid, msg.messageType)
		p.handleReadNoOwner(msg)
	case READPAGECONTENT:
		fmt.Printf("[Processor %d] Received %s message from Processor %d \n", p.pid, msg.messageType, msg.senderID)
		p.handleReadPageContent(msg)
	}
}

func (p *Processor) write(page int, content string) {
	p.waitGroup.Add(1)
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
	msg := <-p.responseChannel
	switch msg.messageType {
	case WRITENOOWNER:
		fmt.Printf("[Processor %d] Received %s message from Central Manager \n", p.pid, msg.messageType)
		p.handleWriteNoOwner(msg)
	case WRITEPAGECONTENT:
		fmt.Printf("[Processor %d] Received %s message from Processor %d \n", p.pid, msg.messageType, msg.senderID)
		p.handleWritePageContent(msg)
	}
}

func main() {

	// Wait group to wait later
	var wg sync.WaitGroup

	// Create central manager
	cm := &CentralManager{
		processors:     make(map[int]*Processor),
		waitGroup:      &wg,
		pageOwner:      make(map[int]int),
		pageCopyset:    make(map[int][]int),
		requestChannel: make(chan Message),
		confirmChannel: make(chan Message),
	}

	// Create processors
	allProcessors := make(map[int]*Processor)
	for i := 1; i <= NUM_PROCESSORS; i++ {
		p := &Processor{
			pid:             i,
			cm:              cm,
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
	cm.processors = allProcessors

	// Give every processor pointers to every other processor
	for _, p1 := range allProcessors {
		for _, p2 := range allProcessors {
			if p1.pid != p2.pid {
				p1.otherProcessors[p2.pid] = p2
			}
		}
	}

	// Central Manager listens for messages from any processor and responds correspondingly
	// All processors listen for messages from Central Manager and other processors
	go cm.listen()
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

}
