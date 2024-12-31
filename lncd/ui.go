package main

import "net/http"

func formHandler(w http.ResponseWriter, r *http.Request) {
	html := `
        <!DOCTYPE html>
        <html>
        <head>
            <title>LNCD</title>
            <script>
                function submitForm(event) {
                    event.preventDefault();
                    const form = event.target;
					const response = document.getElementById('response');
                    const authToken = form.authtoken.value;
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
                            'Content-Type': 'application/json',
                            'Authorization': 'Bearer ' + authToken
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
            <h1>LNCD Test Form</h1>
            <form onsubmit="submitForm(event)">
                <label for="mailbox">AuthToken:</label><br>
                <input value="" type="text" id="authtoken" name="authtoken"><br>
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
