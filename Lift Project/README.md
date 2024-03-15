# Western Elevator
<img src="elevator-1.png" alt="Alt text" width="400"/>

Hold your horses and beware the elevator system from the wild west!. The local Sheriff is orchestarting all of it with good help from his Wranglers. The Sheriff is the master, wranglers are slaves. Additonaly one of the wranglers are appointed deputy, ready to step in at any time should the sherriff fail.

## Getting Started
The elevator system can be run out of the box with `go run main.go`. The system is by default configured to be run with three elevators with 4 floors. To disable process pair use `--disableWatcher=true`. Nodes have IDs based on local ips so for running multiple elevator on the <u>same host</u>, please provide id's to the different nodes. If no id is specified it gets id "0". For the following ids on the same host you must increment with 1 so the second node on the <u>same host</u> would need the flag `--id=1` otherwise nodes will get duplicate ids.

## Config file
To further customize the elevatorsystem for your needs, please modify config.go

## Acknowledgments
Thank you GitHub CoPilot and youtube for sparring. We also acknowledge that a certain lab  has room for  improvement in regards of air quality.

## Packetloss:
use `sudo ./packetloss -p 16000,16569,20000,12345 -r 0.35`
