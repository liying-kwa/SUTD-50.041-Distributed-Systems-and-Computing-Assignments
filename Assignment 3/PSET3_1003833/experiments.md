# 50.041 Assignment 3
Kwa Li Ying (1003833)

### Experiments and their observations
Note:
- Instructions on how to compile and run programs + general explanation are in README.md
- Part 3 explanation is in another file called part3.md

## Experiment 1: Without any faults, compare the performance of the basic version of Ivy protocol and the new fault tolerant version using requests from at least 10 clients. You should assume at least three replicas for the central manager.
- Both versions are run with the following test operations in sequential order:
	1. Each node i reads from page i
	2. Each node i writes to page i
	3. Each node i reads from page i+1
	4. Each node i writes to page i+1
- Assumptions:
	* Time taken for node to win election is 500ms
	* To simulate network latency, messages sent may have a delay of up to 300 milliseconds
- The timings for NUM_CLIENT = 1 to 10 are recorded in the table below

| Number of Clients | Ivy   | Fault Tolerant Ivy |
|-------------------|-------|--------------------|
| 1                 | 1.47  | 2.48               |
| 2                 | 3.04  | 4.10               |
| 3                 | 5.09  | 6.07               |
| 4                 | 6.97  | 8.02               |
| 5                 | 9.23  | 10.32              |
| 6                 | 11.29 | 12.48              |
| 7                 | 13.50 | 14.68              |
| 8                 | 15.68 | 16.76              |
| 9                 | 17.66 | 18.30              |
| 10                | 19.71 | 20.81              |

- The timings for the fault tolerant version are longer than that of the normal version because the fault tolerant version has to run elections for the Central Manager at least once


## Experiment 2: Evaluate the new design in the presence of faults. Specifically, you can simulate two scenarios a) when the primary CM fails at a random time, and b) when the primary CM restarts after the failure. Compare the performance of these two cases with the equivalent scenarios without any CM faults.
- Code is written in the main function of fault_tolerant_ivy.go
- Simulation for both scenarios work as intended and shown in printed output when the program is run

Note that the implementation for fault tolerance is as such:
- The restarted primary server will NOT start an election because it might potentially mess up the ongoing read/write operations. An election will only be started if the current CM fails.
- If the CM fails in between operations then there is no fault tolerance issue and the newly elected replica has the correct metadata.
- If the CM fails before it sends READFORWARD/WRITEFORWARD/READNOOWNER/WRITENOOWNER, then the processor who requested the operation should timeout and redo the read/write operation. No changes have been made to the metadata yet so no problem.
- If the CM fails after it sends READFORWARD/WRITEFORWARD/READNOOWNER/WRITENOOWNER, then the recipient processor will receive the message and continue with the operation. The metadata should be updated on ALL replica so there should not be a fault tolerance issue when the CM position is passed over to the newly elected replica. The newly elected replica will also receive a READCONFIRM/WRITECONFIRM msg but this does not affect anything.




