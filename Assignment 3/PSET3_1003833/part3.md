# 50.041 Assignment 3
Kwa Li Ying (1003833)

### Part 3
Note: 
- Instructions on how to compile and run programs + general explanation are in README.md
- Experiments and their observations are in another file called experiments.md

## Q: Argue whether your design still preserves sequential consistency (a short paragraph will suffice).
- Yes, my fault tolerant design of Ivy still preserves sequential consistency
- The shared memory portion follows sequential consistency <br />
	&rightarrow; Almost exacly the same as the example in class
- The election portion's operations may not exactly be sequential, but does not affect the consistency of the shared memory portion <br />
	&rightarrow; There could be concurrent operations especially when messages are broadcasted
	&rightarrow; Does not affect the consistency of reads and write bcos all read/write operations are temporarily halted when the election is ongoing
- The shared memory leading to election portion follows sequential consistency <br />
	&rightarrow; In the middle of a read/write operation, if the processor realises that the Central Manager is not responding, it sends messages to all other CM replicas to tell them to start an election
- The election leading to shared memory portion also follows sequential consistency <br />
	&rightarrow; While there is no assigned central manager yet, the processor (in the middle of a read/write operation) busy waits with one-second intervals <br />
	&rightarrow; When the election ends and every processor is given the pointer to the central manager, the busy-waiting processor continues its read/write operation


