# Go framework for syslog sidecars creation 
 
[![GoDev](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white)](https://pkg.go.dev/github.com/g41797/syslogsidecar)
[![Go](https://github.com/g41797/syslogsidecar/actions/workflows/go.yml/badge.svg)](https://github.com/g41797/syslogsidecar/actions/workflows/go.yml)

Any **syslogsidecar** based process consists of:
- syslog server and run-time environment provided by syslogsidecar
- broker specific plugins developed in separated repos
	
## syslog server component
syslog server component of sidecar:
  - receives logs intended for [syslogd](https://linux.die.net/man/8/syslogd)
  - parses, validates and filters messages
  - converts messages to easy for further processing  _*partname=partvalue*_ format
  - supports RFCs:
    - [RFC3164](<https://tools.ietf.org/html/rfc3164>)
    - [RFC5424](<https://tools.ietf.org/html/rfc5424>)
    
  
### RFC3164

  RFC3164 is oldest syslog RFC, syslogsidecar supports it for old syslogd clients.

  RFC3164 message consists of following symbolic parts:
  - "priority" (priority = facility * 8 + severity Level)
  - "facility" 
  - "severity"
  - "timestamp"
  - "hostname"
  - "tag"
  - "**content**" (text of the message)

### RFC5424

  RFC5424 message consists of following symbolic parts:
 - "priority" (priority = facility * 8 + severity level)
 - "facility" 
 - "severity"
 - "timestamp"
 - "hostname"
 - "version"
 - "app_name"
 - "proc_id"
 - "msg_id"
 - "structured_data"
 - "**message**" (text of the message)

### Non-RFC parts

  syslogsidecar adds rfc of produced message:
  - Part name: "rfc"
  - Values: "RFC3164"|"RFC5424"

### Badly formatted messages

  syslogsidecar creates only one part for badly formatted message - former syslog message:
  - Part name: "data"
  
### Syslog facilities
The facility represents the machine process that created the Syslog event
| Name | Value | Description |
| :---          |  :---:           |          :--- |
|"kern"      | 0  |     kernel messages |
|"user"      | 1  |     random user-level messages |
|"mail"      | 2  |     mail system |
|"daemon"    | 3  |     system daemons |
|"auth"      | 4  |     security/authorization messages |
|"syslog"    | 5  |     messages generated internally by syslogd |
|"lpr"       | 6  |     line printer subsystem |
|"news"      | 7  |     network news subsystem |
|"uucp"      | 8  |     UUCP subsystem |
|"cron"      | 9  |     clock daemon |
|"authpriv"  | 10 |     security/authorization messages (private) |
|"ftp"       | 11 |     ftp daemon |
|"local0"    | 16 |     local use 0 |
|"local1"    | 17 |     local use 1 |
|"local2"    | 18 |     local use 2 |
|"local3"    | 19 |     local use 3 |
|"local4"    | 20 |     local use 4 |
|"local5"    | 21 |     local use 5 |
|"local6"    | 22 |     local use 6 |
|"local7"    | 23 |     local use 7 |



### Severity levels
   As the name suggests, the severity level describes the severity of the syslog message in question. 

| Level | Name | Description |
| :---:          |  :---           |          :--- |
|0| emerg   |  system is unusable               |
|1| alert   |  action must be taken immediately |
|2| crit    |  critical conditions              |
|3| err     |  error conditions                 |
|4| warning |  warning conditions               |
|5| notice  |  normal but significant condition |
|6| info    |  informational                    |
|7| debug   |  debug-level messages             |

  syslogsidecar filters messages by severity level according to value in configuration, e.g. for
```json
{
  "SEVERITYLEVEL": 4,
}
```
all messages with severity above 4 will be discarded. 

### Timestamp format

syslogsidecar saves timestamps in [RFC3339](https://datatracker.ietf.org/doc/html/rfc3339) format

### Configuration

  Configuration of syslog server component of syslogsidecar is saved in the file syslogreceiver.json:
```json
{
    "SEVERITYLEVEL": 4,
    "ADDRTCP": "127.0.0.1:5141",
    "ADDRUDP": "127.0.0.1:5141",
    "UDSPATH": "",
    "ADDRTCPTLS": "127.0.0.1:5143",
    "CLIENT_CERT_PATH": "",
    "CLIENT_KEY_PATH ": "",
    "ROOT_CA_PATH": ""
}
```
and related go struct:
```go
type SyslogConfiguration struct {
	// The Syslog Severity level ranges between 0 to 7.
	// Each number points to the relevance of the action reported.
	// From a debugging message (7) to a completely unusable system (0):
	//
	//	0		Emergency: system is unusable
	//	1		Alert: action must be taken immediately
	//	2		Critical: critical conditions
	//	3		Error: error conditions
	//	4		Warning: warning conditions
	//	5		Notice: normal but significant condition
	//	6		Informational: informational messages
	//	7		Debug: debug-level messages
	//
	// Log with severity above value from configuration will be discarded
	// Examples:
	// -1 - all logs will be discarded
	// 5  - logs with severities 6(Informational) and 7(Debug) will be discarded
	// 7  - all logs will be processed
	SEVERITYLEVEL int

	// IPv4 address of TCP listener.
	// For empty string - don't use TCP
	// e.g "0.0.0.0:5141" - listen on all adapters, port 5141
	// "127.0.0.1:5141" - listen on loopback "adapter"
	ADDRTCP string

	// IPv4 address of UDP receiver.
	// For empty string - don't use UDP
	// Usually "0.0.0.0:5141" - receive from all adapters, port 5141
	// "127.0.0.1:5141" - receive from loopback "adapter"
	ADDRUDP string

	// Unix domain socket name - actually file path.
	// For empty string - don't use UDS
	// Regarding limitations see https://man7.org/linux/man-pages/man7/unix.7.html
	UDSPATH string

	// TLS section: Listening on non empty ADDRTCPTLS will start only
	// for valid tls configuration (created using last 3 parameters)
	ADDRTCPTLS       string
	CLIENT_CERT_PATH string
	CLIENT_KEY_PATH  string
	ROOT_CA_PATH     string
}
```
## Plugins

There are 3 kinds of broker specific plugins:
- connector
- producer
- consumer (only for tests)

### Connector
- connects to the broker
- periodically validate connection state and re-connect in case of failure
- informs another parts of the process about status of the connection
- provides additional information 

Interface of connector:
```go
// Connector provides possibility for negotiation between sputnik based
// software and external broker process
type Connector interface {
	// Connect to the broker or attach to existing shared
	Connect(cf sputnik.ConfFactory, shared sputnik.ServerConnection) error

	// For shared connection - detach, for own - close
	Disconnect()
}
```

More about connector and underlying software - [sputnik](https://github.com/g41797/sputnik#readme)

Examples of connector:
- [connector for NATS](https://github.com/g41797/syslog2nats/blob/main/connector.go)
- [connector for Memphis](https://github.com/g41797/memphis-protocol-adapter/blob/master/pkg/adapter/connector.go)


### Producer
  - forwards(produces) messages to the broker

Interface of producer:
```go
type MessageProducer interface {
	Connector

	// Translate message to format of the broker and send it
	Produce(msg sputnik.Msg) error
}
```

Examples of producer:
- [producer for NATS](https://github.com/g41797/syslog2nats/blob/main/msgproducer.go)
- [producer for Memphis](https://github.com/g41797/memphis-protocol-adapter/blob/master/pkg/syslog/msgproducer.go)

  

 ## Implementations are based on syslogsidecar

 - syslog for [Memphis](https://memphis.dev) is part of [memphis-protocol-adapter](https://github.com/g41797/memphis-protocol-adapter) project
 - syslog for [NATS](https://nats.io) - [syslog2nats](https://github.com/g41797/syslog2nats)


 ## Automatic startup of the message broker during test/integration

You can use [starter](https://github.com/g41797/sputnik/blob/main/sidecar/starter.go) for automatic start/stop docker containers with broker services.
```go
	stop, _ := sidecar.StartServices()

	defer stop()

	....................................
```

## Dependencies

Production:
- [sputnik](https://github.com/g41797/sputnik)
- fork of [go-syslog](https://github.com/mcuadros/go-syslog)
- fork of [gonfig](https://github.com/tkanos/gonfig)

Tests:
- [srslog](https://github.com/RackSec/srslog)
- [roaring](https://github.com/RoaringBitmap/roaring)
- [EventBus](https://github.com/asaskevich/EventBus)
- [kissngoqueue](https://github.com/g41797/kissngoqueue)