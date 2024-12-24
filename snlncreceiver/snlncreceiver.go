package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"sync"
	"time"

	"encoding/json"
	"net/http"
	"strconv"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/lightninglabs/lightning-node-connect/mailbox"
	"github.com/lightningnetwork/lnd/build"
	"github.com/lightningnetwork/lnd/keychain"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/signal"
	"google.golang.org/grpc"
)

func getEnv(key string, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := getEnv(key, "")
	if value, err := time.ParseDuration(valueStr); err == nil {
		return value
	}
	return defaultValue
}

var (
	LNC_TIMEOUT                  = getEnvAsDuration("LNC_TIMEOUT", 5*time.Minute)
	LNC_LIMIT_ACTIVE_CONNECTIONS = getEnvAsInt("LNC_LIMIT_ACTIVE_CONNECTIONS", 210)
	LNC_STATS_INTERVAL           = getEnvAsDuration("LNC_STATS_INTERVAL", 1*time.Minute)
	LNC_DEBUG                    = getEnv("LNC_DEBUG", "false") == "true"
)

// //////////////////////////////
// DEBUG LOGS for secrets
// Never turn this on in production or it will leak user
// secrets to the stdout, that is undesirable.
var USAFE_LOGS = false

////////////////////////////////

type ConnectionInfo struct {
	Mailbox       string
	PairingPhrase string
	LocalKey      string
	RemoteKey     string
	Status        string
}

type ConnectionKey struct {
	mailbox       string
	pairingPhrase string
}

type Action struct {
	method     string
	payload    string
	onError    func(error)
	onResponse func(ConnectionInfo, string)
}

type Connection struct {
	connInfo     ConnectionInfo
	actions      chan Action
	grpcClient   *grpc.ClientConn
	registry     map[string]func(context.Context, *grpc.ClientConn, string, func(string, error))
	pool         *ConnectionPool
	timeoutTimer *time.Timer
}

type ConnectionPool struct {
	connections map[ConnectionKey]*Connection
	mutex       sync.Mutex
}

type RpcRequest struct {
	Connection ConnectionInfo
	Method     string
	Payload    string
}

type RpcResponse struct {
	Connection ConnectionInfo
	Result     string
}

func NewConnectionPool() *ConnectionPool {
	return &ConnectionPool{
		connections: make(map[ConnectionKey]*Connection),
	}
}

func NewConnection(pool *ConnectionPool, info ConnectionInfo) (*Connection, error) {
	localPriv, remotePub, err := parseKeys(
		info.LocalKey, info.RemoteKey,
	)
	if err != nil {
		return nil, err
	}

	info.LocalKey = hex.EncodeToString(localPriv.Serialize())

	var ecdhPrivKey keychain.SingleKeyECDH = &keychain.PrivKeyECDH{PrivKey: localPriv}
	statusChecker, lndConnect, err := mailbox.NewClientWebsocketConn(
		info.Mailbox, info.PairingPhrase, ecdhPrivKey, remotePub,
		func(key *btcec.PublicKey) error {
			info.RemoteKey = hex.EncodeToString(key.SerializeCompressed())
			return nil
		}, func(data []byte) error {
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	var lndConn *grpc.ClientConn
	lndConn, err = lndConnect()
	if err != nil {
		return nil, err
	}

	var status = statusChecker().String()
	info.Status = status

	var connection *Connection = &Connection{
		connInfo:   info,
		actions:    make(chan Action, 1),
		grpcClient: lndConn,
		registry:   make(map[string]func(context.Context, *grpc.ClientConn, string, func(string, error))),
		pool:       pool,
	}
	lnrpc.RegisterLightningJSONCallbacks(connection.registry)
	return connection, nil
}

func (conn *Connection) runLoop() {
	for req := range conn.actions {
		var method, ok = conn.registry[req.method]
		if ok {
			log.Infof("Executing method: %v", req.method)
			if USAFE_LOGS {
				log.Debugf("Execution: %v %v %v", conn.connInfo, req.method, req.payload)
			}
			method(context.Background(), conn.grpcClient, req.payload, func(resultJSON string, err error) {
				if err != nil {
					req.onError(err)
				} else {
					req.onResponse(conn.connInfo, resultJSON)
				}
			})
		}
	}
}

func (conn *Connection) Close() {
	close(conn.actions)
	conn.grpcClient.Close()
}

func (pool *ConnectionPool) execute(info ConnectionInfo, req Action) {
	try := func() bool {
		var key ConnectionKey = ConnectionKey{info.Mailbox, info.PairingPhrase}
		var err error
		connection, ok := pool.connections[key]
		if !ok {
			log.Infof("Creating new connection")
			if USAFE_LOGS {
				log.Debugf("Connection: %v", info)
			}
			if len(pool.connections) >= LNC_LIMIT_ACTIVE_CONNECTIONS {
				req.onError(fmt.Errorf("too many active connections"))
				return true
			}
			connection, err = NewConnection(pool, info)
			if err != nil {
				req.onError(err)
			} else {
				connection.timeoutTimer = time.AfterFunc(LNC_TIMEOUT, func() {
					pool.mutex.Lock()
					if len(connection.actions) == 0 {
						log.Infof("Closing idle connection %v", info.RemoteKey)
						if USAFE_LOGS {
							log.Debugf("Connection: %v", info)
						}
						connection.Close()
						delete(pool.connections, ConnectionKey{info.Mailbox, info.PairingPhrase})
					} else {
						connection.timeoutTimer.Reset(LNC_TIMEOUT)
					}
					pool.mutex.Unlock()
				})
				pool.connections[key] = connection
				go connection.runLoop()
			}
		} else {
			log.Infof("Reusing existing connection")
			if USAFE_LOGS {
				log.Debugf("Connection: %v", info)
			}
		}
		connection.actions <- req
		return false
	}

	retry := true
	for retry {
		pool.mutex.Lock()
		retry = try()
		pool.mutex.Unlock()
		if retry {
			log.Infof("Retrying connection")
			time.Sleep(1 * time.Second)
		}
	}
}

func rpcHandler(pool *ConnectionPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
			return
		}
		var request RpcRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		done := make(chan struct{})

		log.Infof("Incoming RPC request: %v", request.Method)
		if USAFE_LOGS {
			log.Debugf("Full request: %v", request)
		}

		var response RpcResponse
		pool.execute(request.Connection, Action{
			method:  request.Method,
			payload: request.Payload,
			onError: func(err error) {
				log.Errorf("RPC error: %v", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				close(done)
			},
			onResponse: func(info ConnectionInfo, result string) {
				log.Infof("RPC response: %v", result)
				if USAFE_LOGS {
					log.Debugf("Connection: %v", info)
				}
				response = RpcResponse{
					Connection: info,
					Result:     result,
				}
				close(done)
			},
		})

		<-done
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func formHandler(w http.ResponseWriter, r *http.Request) {
	html := `
        <!DOCTYPE html>
        <html>
        <head>
            <title>RPC Form</title>
            <script>
                function submitForm(event) {
                    event.preventDefault();
                    const form = event.target;
					const response = document.getElementById('response');
                    const data = {
                        Connection: {
                            Mailbox: form.mailbox.value,
                            PairingPhrase: form.pairingPhrase.value,
                            LocalKey: form.localKey.value,
                            RemoteKey: form.remoteKey.value
                        },
                        Method: form.method.value,
                        Payload: form.payload.value
                    };
                    fetch('/rpc', {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json'
                        },
                        body: JSON.stringify(data)
                    })
                    .then(response => response.json())
                    .then(data => {
                        console.log('Success:', data);
						response.innerHTML = JSON.stringify(data, null, 2);
                    })
                    .catch((error) => {
                        console.error('Error:', error);
						response.innerHTML = error;
                    });
                }
            </script>
			<style>
				input,textarea{ 
				 	width: 90vw;
				}
				textarea {
					height: 20vh;
				}
			</style>
        </head>
        <body>
            <h1>RPC Form</h1>
            <form onsubmit="submitForm(event)">
                <label for="mailbox">Mailbox:</label><br>
                <input value="mailbox.terminal.lightning.today:443" type="text" id="mailbox" name="mailbox"><br>
                <label for="pairingPhrase">Pairing Phrase:</label><br>
                <input type="text" id="pairingPhrase" name="pairingPhrase"><br>
                <label for="localKey">Local Key:</label><br>
                <input type="text" id="localKey" name="localKey"><br>
                <label for="remoteKey">Remote Key:</label><br>
                <input type="text" id="remoteKey" name="remoteKey"><br>
                <label for="method">Method:</label><br>
                <input value="lnrpc.Lightning.AddInvoice" type="text" id="method" name="method"><br>
                <label for="payload">Payload:</label><br>
                <textarea  id="payload" name="payload">{"memo":"test","value":1000}</textarea><br>
                <input type="submit" value="Submit">
            </form>
			<pre id="response"></pre>
        </body>
        </html>
    `
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func parseKeys(localPrivKey, remotePubKey string) (
	*btcec.PrivateKey, *btcec.PublicKey, error) {

	var (
		localStaticKey  *btcec.PrivateKey
		remoteStaticKey *btcec.PublicKey
	)
	switch {

	// This is a new session for which a local key has not yet been derived,
	// so we generate a new key.
	case localPrivKey == "" && remotePubKey == "":
		privKey, err := btcec.NewPrivateKey()
		if err != nil {
			return nil, nil, err
		}
		localStaticKey = privKey
		if USAFE_LOGS {
			log.Debugf("Generated new priv key: %v", hex.EncodeToString(privKey.Serialize()))
		}

	// A local private key has been provided, so parse it.
	case remotePubKey == "":
		privKeyByte, err := hex.DecodeString(localPrivKey)
		if err != nil {
			return nil, nil, err
		}
		privKey, _ := btcec.PrivKeyFromBytes(privKeyByte)
		localStaticKey = privKey
		if USAFE_LOGS {
			log.Debugf("Parsed local priv key: %v", hex.EncodeToString(privKey.Serialize()))
		}

	// Both local private key and remote public key have been provided,
	// so parse them both into the appropriate types.
	default:
		// Both local and remote are set.
		localPrivKeyBytes, err := hex.DecodeString(localPrivKey)
		if err != nil {
			return nil, nil, err
		}
		privKey, _ := btcec.PrivKeyFromBytes(localPrivKeyBytes)
		localStaticKey = privKey

		remoteKeyBytes, err := hex.DecodeString(remotePubKey)
		if err != nil {
			return nil, nil, err
		}

		remoteStaticKey, err = btcec.ParsePubKey(remoteKeyBytes)
		if err != nil {
			return nil, nil, err
		}

		if USAFE_LOGS {
			log.Debugf("Parsed local priv key: %v", hex.EncodeToString(privKey.Serialize()))
			log.Debugf("Parsed remote pub key: %v", hex.EncodeToString(remoteStaticKey.SerializeCompressed()))
		}
	}

	return localStaticKey, remoteStaticKey, nil
}

func exit(err error) {
	fmt.Printf("Error running daemon: %v\n", err)
	os.Exit(1)
}

func stats(pool *ConnectionPool) {
	ticker := time.NewTicker(LNC_STATS_INTERVAL)
	go func() {
		for range ticker.C {
			pool.mutex.Lock()
			numConnections := len(pool.connections)
			log.Infof("Number of active connections: %d", numConnections)

			index := 0
			for _, conn := range pool.connections {
				log.Infof("Connection %d", index)
				pendingActions := len(conn.actions)
				log.Infof("    Pending actions: %d", pendingActions)
				log.Infof("    Connection status: %v", conn.connInfo.Status)
				index++
			}
			pool.mutex.Unlock()
		}
	}()

}

func main() {
	shutdownInterceptor, err := signal.Intercept()
	if err != nil {
		exit(err)
	}
	logWriter := build.NewRotatingLogWriter()
	SetupLoggers(logWriter, shutdownInterceptor, LNC_DEBUG)

	log.Infof("Starting daemon")
	log.Infof("LNC_TIMEOUT: %v", LNC_TIMEOUT)
	log.Infof("LNC_LIMIT_ACTIVE_CONNECTIONS: %v", LNC_LIMIT_ACTIVE_CONNECTIONS)
	log.Infof("LNC_STATS_INTERVAL: %v", LNC_STATS_INTERVAL)
	log.Infof("LNC_DEBUG: %v", LNC_DEBUG)

	var pool *ConnectionPool = NewConnectionPool()
	stats(pool)

	http.HandleFunc("/rpc", rpcHandler(pool))
	http.HandleFunc("/", formHandler)

	log.Infof("Server started at :7167")
	if err := http.ListenAndServe(":7167", nil); err != nil {
		log.Errorf("Error starting server: %v", err)
		exit(err)
	}

	<-shutdownInterceptor.ShutdownChannel()
	log.Infof("Shutting down daemon")
	for _, conn := range pool.connections {
		conn.Close()
	}
	log.Infof("Shutdown complete")

}
