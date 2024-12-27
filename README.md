# LNC*Daemonized* 

This is a Golang daemon that exposes lnc methods through a REST-like API, serving as a server-side alternative to lnc-web.

Currently, it supports only the `lnrpc.Lightning methods` and `checkPerms`.
Additional methods can be easily registered in `lncd.go`.

Lifecycle of LNC connections is managed. Connections are reused whenever possible and are automatically terminated after a period of inactivity.

Configuration options can be specified via environment variables (all are optional):


| Environment Variable    | Default Value   | Description                                                                 |
|-------------------------|-----------------|-----------------------------------------------------------------------------|
| `LNCD_TIMEOUT`           | `5m` | Timeout duration for connections.                                           |
| `LNCD_LIMIT_ACTIVE_CONNECTIONS` | `210`           | Maximum number of active connections allowed.                               |
| `LNCD_STATS_INTERVAL`    | `1m` | Interval for logging connection pool statistics.                            |
| `LNCD_DEBUG`             | `true`         | Flag to enable or disable debug logging.                                    |
| `LNCD_RECEIVER_PORT`     | `7167`          | Port on which the receiver server listens.                                  |
| `LNCD_RECEIVER_HOST`     | `0.0.0.0`       | Host address on which the receiver server listens.                          |



## Intended scope

This was developed to serve as the backend for stacker.news' LNC receive attachments. 
It has only been tested with the `lnrpc.Lightning.AddInvoice` method, so if you plan to use this as a general-purpose LNC connector, ensure you test and review it accurately.


## Usage example

You can test the commands below using the web ui at http://localhost:7167/ or by sending POST requests to http://localhost:7167/rpc

```
POST /rpc
{
    "Connection":{
        "Mailbox": "mailbox.terminal.lightning.today:443",
        "PairingPhrase": "...."        
    },
	"Method": "checkPerms"
	"Payload": "[\"lnrpc.Lightning.AddInvoice\", \"lnrpc.Lightning.SendPaymentSync\"]" 
}

RESPONSE 
{
  "Connection": {
    "Mailbox": "mailbox.terminal.lightning.today:443",
    "PairingPhrase": "...",
    "LocalKey": "...", // put this back into the next request to reuse the same pairing phrase
    "RemoteKey": "...", // put this back into the next request to reuse the same pairing phrase
    "Status": "Connected"
  },
  "Result": "[true,false]"
}

```

```
POST /rpc
{
    "Connection":{
        "Mailbox": "mailbox.terminal.lightning.today:443",
        "PairingPhrase": "...."        
    },
	"Method": "lnrpc.Lightning.AddInvoice"
	"Payload": "{\"memo\":\"test\",\"value\":1000}"
}


RESPONSE
{
  "Connection": {
    "Mailbox": "mailbox.terminal.lightning.today:443",
    "PairingPhrase": "...",
    "LocalKey": "...", // put this back into the next request to reuse the same pairing phrase
    "RemoteKey": "...", // put this back into the next request to reuse the same pairing phrase
    "Status": "Connected"
  },
  "Result": "{\"r_hash\":\"xx\", \"payment_request\":\"xx\", \"add_index\":\"x\", \"payment_addr\":\"xx\"}"
}


```