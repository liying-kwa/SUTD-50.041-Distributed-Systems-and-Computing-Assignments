
# Performance Timings

Note: All timings are in seconds

Conditions/Assumptions
* Critical section duration is 1 second
* To simulate network latency, messages sent may have a delay of up to 300 milliseconds


| Number of Nodes | Lamport SPQ | Ricart and Agrawala | Centralised Lock Server |
|-----------------|-------------|---------------------|-------------------------|
| 1               | 1.00        | 1.00                | 1.42                    |
| 2               | 2.53        | 2.21                | 2.41                    |
| 3               | 3.66        | 2.68                | 3.76                    |
| 4               | 4.91        | 3.84                | 5.11                    |
| 5               | 6.18        | 4.98                | 6.20                    |
| 6               | 7.21        | 7.09                | 7.62                    |
| 7               | 8.47        | 8.45                | 9.06                    |
| 8               | 9.37        | 8.70                | 10.17                   |
| 9               | 10.67       | 9.41                | 11.46                   |
| 10              | 11.23       | 9.64                | 12.71                   |
