# LNC*Daemonized* 

This is a Golang daemon that exposes lnc methods through a REST-like API, serving as a server-side alternative to lnc-web.

Currently, it supports only the `lnrpc.Lightning methods`. 
Additional methods can be easily registered in `lncd.go`.

Lifecycle of LNC connections is managed. Connections are reused whenever possible and are automatically terminated after a period of inactivity.

Configuration options can be specified via environment variables (all are optional):


| Environment Variable    | Default Value   | Description                                                                 |
|-------------------------|-----------------|-----------------------------------------------------------------------------|
| `LNCD_TIMEOUT`           | `5*time.Minute` | Timeout duration for connections.                                           |
| `LNCD_LIMIT_ACTIVE_CONNECTIONS` | `210`           | Maximum number of active connections allowed.                               |
| `LNCD_STATS_INTERVAL`    | `1*time.Minute` | Interval for logging connection pool statistics.                            |
| `LNCD_DEBUG`             | `false`         | Flag to enable or disable debug logging.                                    |
| `LNCD_RECEIVER_PORT`     | `7167`          | Port on which the receiver server listens.                                  |
| `LNCD_RECEIVER_HOST`     | `0.0.0.0`       | Host address on which the receiver server listens.                          |



## Intended scope

This was developed to serve as the backend for stacker.news' LNC receive attachments. 
It has only been tested with the `lnrpc.Lightning.AddInvoice` method, so if you plan to use this as a general-purpose LNC connector, ensure you test and review it accurately.

