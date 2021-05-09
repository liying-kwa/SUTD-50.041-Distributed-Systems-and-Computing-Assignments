# 50.041 Assignment 2
Kwa Li Ying (1003833)

### Note: Performance timings are in another file called performance.md

## Q1: Lamport's SPQ without Ricart and Agrawala's optimisation

- The folder for this is 'lamport_spq'
- There is only one go file, called 'lamport_spq.go'
- To run the file, cd into the folder and use the command 'go run .'
- Most explanations are in the output (from running the code) or in comments in the code
- In the main function, each node requests to enter the critical section once
- To test the logical clock component, each node makes a second request to enter the critical section. This is commented out by default.
- At the end of the main function, the time taken for the last node to exit the critical section is printed.
- NUM_NODES, CS_DURATION and MAX_SEND_DURATION can be varied by editing these constants at the top of the code. NUM_NODES indicate the number of nodes spawned at the beginning, CS_DURATION indicate the time taken to execute the critical section (in seconds), and MAX_SEND_DURATION is the maximum delay (in milliseconds) that a node takes before it sends out a message to another node


## Q2: Lamport's SPQ with Ricart and Agrawala's optimisation
- The folder for this is 'ricart_and_agrawala'
- There is only one go file, called 'ricart_and_agrawala.go'
- To run the file, cd into the folder and use the command 'go run .'
- Most explanations are in the output (from running the code) or in comments in the code
- In the main function, each node requests to enter the critical section once
- To test the logical clock component, each node makes a second request to enter the critical section. This is commented out by default.
- At the end of the main function, the time taken for the last node to exit the critical section is printed.
- NUM_NODES, CS_DURATION and MAX_SEND_DURATION can be varied by editing these constants at the top of the code. NUM_NODES indicate the number of nodes spawned at the beginning, CS_DURATION indicate the time taken to execute the critical section (in seconds), and MAX_SEND_DURATION is the maximum delay (in milliseconds) that a node takes before it sends out a message to another node


## Q3: Centralised Server Protocol
- The folder for this is 'centralised_server'
- There is only one go file, called 'centralised_server.go'
- To run the file, cd into the folder and use the command 'go run .'
- Most explanations are in the output (from running the code) or in comments in the code
- In the main function, each node requests to enter the critical section once
- At the end of the main function, the time taken for the last node to exit the critical section is printed.
- NUM_NODES, CS_DURATION and MAX_SEND_DURATION can be varied by editing these constants at the top of the code. NUM_NODES indicate the number of nodes spawned at the beginning, CS_DURATION indicate the time taken to execute the critical section (in seconds), and MAX_SEND_DURATION is the maximum delay (in milliseconds) that the server takes before it sends out a message to a node and a node takes before it sends out a message to the server



