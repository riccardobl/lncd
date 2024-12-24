# LNC *daemonized* Receiver 

This is a Golang daemon that exposes lnc methods through a REST-like API, serving as a server-side alternative to lnc-web.

Currently, it supports only the `lnrpc.Lightning methods`. 
Additional methods can be easily registered in `snlncreceiver.go`.

Lifecycle of LNC connections is managed. Connections are reused whenever possible and are automatically terminated after a period of inactivity.

Configuration options can be specified via environment variables (all are optional):


| Environment Variable    | Default Value   | Description                                                                 |
|-------------------------|-----------------|-----------------------------------------------------------------------------|
| `LNC_TIMEOUT`           | `5*time.Minute` | Timeout duration for connections.                                           |
| `LNC_LIMIT_ACTIVE_CONNECTIONS` | `210`           | Maximum number of active connections allowed.                               |
| `LNC_STATS_INTERVAL`    | `1*time.Minute` | Interval for logging connection pool statistics.                            |
| `LNC_DEBUG`             | `false`         | Flag to enable or disable debug logging.                                    |
| `LNC_RECEIVER_PORT`     | `7167`          | Port on which the receiver server listens.                                  |
| `LNC_RECEIVER_HOST`     | `0.0.0.0`       | Host address on which the receiver server listens.                          |