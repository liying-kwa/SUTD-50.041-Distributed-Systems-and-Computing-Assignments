package main

import (
	"fmt"
	"time"
)

func main() {

	fmt.Println("==========================")
	fmt.Println("      Initialization      ")
	fmt.Println("==========================")

	// Create and start nodes
	var allNodes []*Node
	for i := 0; i < NUM_NODES; i++ {
		n := createNode(i)
		allNodes = append(allNodes, n)
	}
	for i := 0; i < NUM_NODES; i++ {
		go allNodes[i].start(allNodes)
	}

	// Testing initialisation (including first election)
	time.Sleep(time.Second * 5)
	fmt.Println()
	fmt.Println("Initialization election results:")
	for pid, node := range allNodes {
		fmt.Printf("Node %d has electionStatus %s and coordinator %d \n", pid, node.election.status, node.election.coordinatorPid)
	}

	fmt.Println()
	time.Sleep(time.Second)

	fmt.Println("======================================")
	fmt.Println("          Part 1: Worst Case          ")
	fmt.Println("======================================")

	// Kill highest pid node and make node 0 start an election.
	fmt.Printf("Node %d fails. \n", NUM_NODES-1)
	allNodes[NUM_NODES-1].killChannel <- true
	time.Sleep(time.Second)
	fmt.Println("Node 0 discovers this and starts an election.")
	go allNodes[0].startElection()

	// Worst case election results
	time.Sleep(time.Second * 5)
	fmt.Println()
	fmt.Println("Worst case election results:")
	for pid, node := range allNodes {
		fmt.Printf("Node %d has electionStatus %s and coordinator %d \n", pid, node.election.status, node.election.coordinatorPid)
	}

	fmt.Println()
	time.Sleep(time.Second)

	// Revive Node with highest pid
	fmt.Printf("Worst case done, so reviving Node %d... \n", NUM_NODES-1)
	go allNodes[NUM_NODES-1].start(allNodes)

	// Testing revival
	time.Sleep(time.Second * 5)
	fmt.Println()
	fmt.Println("Revival election results:")
	for pid, node := range allNodes {
		fmt.Printf("Node %d has electionStatus %s and coordinator %d \n", pid, node.election.status, node.election.coordinatorPid)
	}

	fmt.Println()
	time.Sleep(time.Second)

	fmt.Println("=====================================")
	fmt.Println("          Part 1: Best Case          ")
	fmt.Println("=====================================")

	// Kill highest pid node and make node with 2nd highest pid start an election.
	fmt.Printf("Node %d fails. \n", NUM_NODES-1)
	allNodes[NUM_NODES-1].killChannel <- true
	time.Sleep(time.Second)
	fmt.Printf("Node %d discovers this and starts an election. \n", NUM_NODES-2)
	go allNodes[NUM_NODES-2].startElection()

	// Best case election results
	time.Sleep(time.Second * 5)
	fmt.Println()
	fmt.Println("Best case election results:")
	for pid, node := range allNodes {
		fmt.Printf("Node %d has electionStatus %s and coordinator %d \n", pid, node.election.status, node.election.coordinatorPid)
	}

	fmt.Println()
	time.Sleep(time.Second)

	fmt.Println("========================================")
	fmt.Println("          Part 2: Special Case          ")
	fmt.Println("========================================")

	// Highest pid node is already dead
	fmt.Printf("Node %d is already dead from the previous scenario. \n", NUM_NODES-1)

	// Start election in one node and kill node with 2nd highest pid in the middle of the election
	fmt.Printf("Node 1 discovers Node %d is dead. It starts an election. \n", NUM_NODES-1)
	go allNodes[1].startElection()
	time.Sleep(time.Millisecond * 100)
	fmt.Printf("Assume Node %d fails. Killing Node %d... \n", NUM_NODES-2, NUM_NODES-2)
	allNodes[NUM_NODES-2].killChannel <- true

	// Special case election results
	time.Sleep(time.Second * 5)
	fmt.Println()
	fmt.Println("Special case election results:")
	for pid, node := range allNodes {
		fmt.Printf("Node %d has electionStatus %s and coordinator %d \n", pid, node.election.status, node.election.coordinatorPid)
	}

	// Note about special case
	fmt.Println()
	time.Sleep(time.Second)
	fmt.Println("Note: If Node 3 fails before it rejects node 2, then Node 2 will be the elected coordinator. " +
		"However, if Node 3 fails after it announces itself as the coordinator, then Node 3 will be the elected coordinator. " +
		"In this case, in the real bully algorithm, sooner or later a Node will realise this and start an election. " +
		"This fault detection is left out in this code for simplicity.")

	fmt.Println()
	time.Sleep(time.Second * 5)
	fmt.Println("============================================")
	fmt.Println("       Part 3: Simultaneous Elections       ")
	fmt.Println("============================================")

	// Part 3 is already demonstrated
	fmt.Println()
	time.Sleep(time.Second)
	fmt.Println("Part 3 is already demonstrated in the beginning " +
		"where all nodes start election simultaneously after being created.")
	fmt.Println()

}
