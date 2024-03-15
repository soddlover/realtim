# Western Elevator
![Alt text](elevator-1.png?raw=true "")

Hold your horses and beware the elevator system from the wild west! It is powerd by 15 horses driving a huge horsemill. The local Sheriff is orchestarting all of it with good help from his Wranglers who are ready to step in at any time.

## Getting Started
The elevator system can be run out of the box with go run main.go. The system is by default configured to be run with three elevators with 4 floors. To disable process pair use -disableWatcher=true. For running multiple elevator on the same host, please provide id's to the diffrent elevators on the same hos, for instance -id=0 and -id=1.

## Config file
To further customize the elevatorsystem for your needs, please modify config.go

## Acknoledgments
Thank you GitHub CoPilot for sparring. Sanntidsalen has room for improvement in regards of air quality.

## Packetloss:
use sudo ./packetloss -p 16000,16569,20000,12345 -r 0.35
