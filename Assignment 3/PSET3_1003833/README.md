# 50.041 Assignment 3
Kwa Li Ying (1003833)

### How to compile and run programs + general explanation
Note: 
- Part 3 explanation is in another file called part3.md
- Experiments and their observations are in another file called experiments.md

## Q1: Ivy architecture discussed in class
- The folder for this is 'ivy'.
- There is only one go file, called 'ivy.go'.
- To run the file, cd into the folder and use the command 'go run .'.
- Most explanations are in the output (from running the code) or in comments in the code.
- To read a page from a processor, in the main function, use allProcessors[pid].read(pageNumber).
- To write to a page from a processor, in the main function, use allProcessors[pid].write(pageNumber, pageContentToWrite).
- Some test cases are present in the main function (before Experiment 1). Uncomment to see the output. The test cases are as follows:
	1. PID 2 read(3)
	2. PID 1 write(3, "This is written by pid 1")
	3. PID 3 read(3)
	4. PID 2 read(3)
	5. PID 2 write(3, "This is written by pid 2")
	6. PID 1 read(3)
	7. PID 3 read(3)
- NUM_NODES, CS_DURATION and MAX_SEND_DURATION can be varied by editing these constants at the top of the code. NUM_NODES indicate the number of nodes spawned at the beginning, CS_DURATION indicate the time taken to execute the critical section (in seconds), and MAX_SEND_DURATION is the maximum delay (in milliseconds) that a node takes before it sends out a message to another node
- NUM_PROCESSORS, NUM_REPLICA, NUM_PAGES, MAX_SEND_DURATION, MSG_TIMEOUT and ELECTIONTIMEOUT can be varied by editing these constants at the top of the code. Details about the constants are as follows:
	* NUM_PROCESSORS indicate the number of processors spawned at the beginning
	* NUM_REPLICA indicate the number of replica of the Central Manager, including both primary and secondary replica
	* NUM_PAGES indicate the maximum page number
	* MAX_SEND_DURATION is the maximum delay (in milliseconds) that the central manager / a processor takes before it sends out a message
	* MSG_TIMEOUT is the maximum time (in seconds) that a processor waits for a reply before it deems that the central manager has failed
	* ELECTION_TIMEOUT is the maximum time (in milliseconds) that a CM replica waits for rejection messages before it declares itself the winner of an election
- Experiments are written in the main function and observations are recorded in the experiment.md file


## Q2: Fault tolerant version of Ivy architecture
- The folder for this is 'fault_tolerant_ivy'.
- There is only one go file, called 'fault_tolerant_ivy.go'.
- To run the file, cd into the folder and use the command 'go run .'.
- Most explanations are in the output (from running the code) or in comments in the code.
- To write to a page from a processor, in the main function, use allProcessors[pid].write(pageNumber, pageContentToWrite).
- Some test cases are present in the main function (before Experiment 1). Uncomment to see the output. The test cases are as follows:
	1. PID 2 read(3)
	2. PID 1 write(3, "This is written by pid 1")
	3. PID 3 read(3)
	4. PID 2 read(3)
	5. PID 2 write(3, "This is written by pid 2")
	6. PID 1 read(3)
	7. PID 3 read(3)
- NUM_PROCESSORS, NUM_PAGES and MAX_SEND_DURATION can be varied by editing these constants at the top of the code. Details about the constants are as follows:
	* NUM_PROCESSORS indicate the number of processors spawned at the beginning
	* NUM_PAGES indicate the maximum page number
	* MAX_SEND_DURATION is the maximum delay (in milliseconds) that the central manager / a processor takes before it sends out a message
- Experiments are written in the main function and observations are recorded in the experiment.md file


Note that the implementation for fault tolerance is as such:
- If the CM fails in between operations then there is no fault tolerance issue and the newly elected replica has the correct metadata
- If the CM fails before it sends READFORWARD/WRITEFORWARD/READNOOWNER/WRITENOOWNER, then the processor who requested the operation should timeout and redo the read/write operation. No changes have been made to the metadata yet so no problem
- If the CM fails after it sends READFORWARD/WRITEFORWARD/READNOOWNER/WRITENOOWNER, then the recipient processor will receive the message and continue with the operation. The metadata should be updated on ALL replica so there should not be a fault tolerance issue when the CM position is passed over to the newly elected replica. The newly elected replica will also receive a READCONFIRM/WRITECONFIRM msg but this does not affect anything



