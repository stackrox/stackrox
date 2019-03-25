package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/stackrox/rox/pkg/sync"
)

type message struct {
	Headers map[string][]string    `json:"headers"`
	Data    map[string]interface{} `json:"data"`
}

var (
	lock       sync.Mutex
	dataPosted []message

	serverCRT = `-----BEGIN CERTIFICATE-----
MIIDgDCCAmgCCQDYOU2KIlcBQjANBgkqhkiG9w0BAQsFADCBgTELMAkGA1UEBhMC
VVMxCzAJBgNVBAgMAkNBMQswCQYDVQQHDAJTRjERMA8GA1UECgwIc3RhY2tyb3gx
HzAdBgNVBAMMFndlYmhvb2tzZXJ2ZXIuc3RhY2tyb3gxJDAiBgkqhkiG9w0BCQEW
FXN0YWNrcm94QHN0YWNrcm94LmNvbTAeFw0xOTAzMjMxNTQzMjVaFw0yOTAzMjAx
NTQzMjVaMIGBMQswCQYDVQQGEwJVUzELMAkGA1UECAwCQ0ExCzAJBgNVBAcMAlNG
MREwDwYDVQQKDAhzdGFja3JveDEfMB0GA1UEAwwWd2ViaG9va3NlcnZlci5zdGFj
a3JveDEkMCIGCSqGSIb3DQEJARYVc3RhY2tyb3hAc3RhY2tyb3guY29tMIIBIjAN
BgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAuPzgVGykTALNHDljDiCjwI4ZfF2r
lGKWdtvUhurh42Cl2Kfn0Vgy7mYRjdK/uOiSIl6LVXuNw7w4yg48dXm8By+I3+hs
vMH4ixykWxPn6Ez3Utuuwggn/yAs4kE2Wj0ztFMpRHBGL7Qi7oEv+Vo4349ZJg16
a55db45O3LgOED119F1hQvxblNZhcA2hnNOhveXsJLfdOQKz6UA4KtdBFXxEeZuB
fC45wCHw6kjRrBEPYKB4py4ywYMdUHqswBDn6B3LtwvrrJVPTySK4sgZmOTF2XGg
JRm52MS0rYEvBpEtgkPdknoIv0VnxihMUuRhMXHfGOTFyhWuf/nF2aihXwIDAQAB
MA0GCSqGSIb3DQEBCwUAA4IBAQCYT7jo6Durhx+liRgYNO3G3mRyc36syVNGsllU
Mf5wOUHjxWfppWHzldxMeZRKksrg7xfMXdcGaOOZgD8Ir/pPK2HP48g6KIDWCiVO
kh9AGCLY9osxkBqAihtvJWNkEda+wA9ggF/7wx+0Ci+b/1NvXHeNU3uO3rP7Npwc
rxhvyNqv7MwqpMN6V8hFxqM/3ny8aoUedFsYsEvm8Dm1VLyBiIqZk0CA2oj3NIjb
ObOdSTZUQI4TZOXOpJCpa97CnqroNi7RrT05JOfoe/DPmhoJmF4AUrnd/YUb8pgF
/jvC1xBvPVtJFbYeBVysQCrRk+f/NyyUejQv+OCJ+B1KtJh4
-----END CERTIFICATE-----`

	serverKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEogIBAAKCAQEAuPzgVGykTALNHDljDiCjwI4ZfF2rlGKWdtvUhurh42Cl2Kfn
0Vgy7mYRjdK/uOiSIl6LVXuNw7w4yg48dXm8By+I3+hsvMH4ixykWxPn6Ez3Utuu
wggn/yAs4kE2Wj0ztFMpRHBGL7Qi7oEv+Vo4349ZJg16a55db45O3LgOED119F1h
QvxblNZhcA2hnNOhveXsJLfdOQKz6UA4KtdBFXxEeZuBfC45wCHw6kjRrBEPYKB4
py4ywYMdUHqswBDn6B3LtwvrrJVPTySK4sgZmOTF2XGgJRm52MS0rYEvBpEtgkPd
knoIv0VnxihMUuRhMXHfGOTFyhWuf/nF2aihXwIDAQABAoIBAA6wZc/OYK14e3kG
RYtVpCsiHXv2pC1ANtpxUkr0U8OOZRzzGEFTU5gCmto8JeU08oWwJDhBe1xTkd7Z
iot5gyi+6Yt+FURX0riZKaPhzMRxeCIoN5RIuYRUtbuxmfNIcNac0+YPVENtdmih
8YFHXMTDyTxwTYxGIv08u55jLxqy5B2UyWsNapfSmJQSelvC061B0rfrqCsosTbS
kWQBpACshyF2lXD06nXyRIQrA3v3J+If4gaONz6z/2+IexfBfPhuM0WEf0HTtguQ
4MWFso1eaYGh1juidIrN+oRCN0WB47BN+k5gTVWa4BVDJoP95ebT1hpO3sx1aYm6
UK++HiECgYEA3h8yoV6KkVmTsAfRMMXfOD8n7q2qhMX3prYhwPRuNxFNdRYm7R5D
d86wLFH4ZqFe1KcgYfPZFUVXlRHlVgjQRS3MkM7gT5S1zn/VTjWZYov6ne+jCRg5
/NWE+d1pTNeJmcZSQewWcLM9L9xpcEuwBUdUf7X2CRNiWrV6h9FhiDUCgYEA1TPF
EV0fdVynOT7DgN1j0AJPAxXtRG/aO9vDWZH08YC2l5qp9WXeY77Wlze468fcWJIo
UjPQAkkL8YpYNdfjVIPaMqgIvjeao6yIrYFgsP8X3L6MrZtNhztm0oMemM0tTGUV
4eFK0J+ZvxalrwJ2LzYuDWyGyYluwKXnm/zsfcMCgYBgJl8TTUpsSrtMesXJ+A19
WpFdlx12Jf/i0Xpg/S3sdnfyFCm7gNsxtG28cas2OepD4Sh6XkT9GSwlYj7E9EG7
gGzJzlN4/2WHwvxBw5/m8bMFxOLtH+iSEpdiVb6sPazZvOiEkr7QADafTijyLEFA
t7TTJ6AeI57ypxYoTrGKdQKBgFNni0J9sZ7R/kEwwn6ZHUD0hkBoxYcuUqt0D3ns
1WvctJGeWbq8fUF8GKrTi64BY7vqgYeW6VrbhKabPmLh7/bSFfwXLERtsDszdcya
fl7/jDA5AwOva6bpoBHeZYvVSFFIgkT5Q7FVnmnYzDwotF9HzMBHonsZHpCS1oZ5
bXLNAoGAX0VIqnuFYDGceT+xfeH6V4TfbR8Cl7Fd3HVatq0xBVaRVcGfJLl+Njne
4u11cU0tZtZT0Td1rERGO2D/sCqwfq3yr+Uw5OEHgS/j2UMLqU549YPy+7mBczlK
gvOgnwEXu7OARYy63osxzjvcnovtbg7Iu6uxuFo5xfIKXlZlfm8=
-----END RSA PRIVATE KEY-----`
)

func postHandler(w http.ResponseWriter, r *http.Request) {
	defer func() {
		_ = r.Body.Close()
	}()
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	dataMap := make(map[string]interface{})
	if err := json.Unmarshal(data, &dataMap); err != nil {
		log.Printf("Error unmarshalling data: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	lock.Lock()
	defer lock.Unlock()
	dataPosted = append(dataPosted, message{Headers: r.Header, Data: dataMap})
	w.WriteHeader(http.StatusOK)
}

func getHandler(w http.ResponseWriter, r *http.Request) {
	lock.Lock()
	defer lock.Unlock()
	resp, err := json.Marshal(&dataPosted)
	if err != nil {
		log.Printf("Failed to marshal resp: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(resp); err != nil {
		log.Printf("Failed to write resp: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getHandler(w, r)
	case http.MethodPost:
		postHandler(w, r)
	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}

func tlsServer() {
	err := http.ListenAndServeTLS(":8443", "server.crt", "server.key", http.HandlerFunc(rootHandler))
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func nonTLSServer() {
	if err := http.ListenAndServe(":8080", http.HandlerFunc(rootHandler)); err != nil {
		panic(err)
	}
}

func writeCerts() {
	if err := ioutil.WriteFile("server.crt", []byte(serverCRT), 777); err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile("server.key", []byte(serverKey), 777); err != nil {
		panic(err)
	}
}

func main() {
	writeCerts()
	go tlsServer()
	go nonTLSServer()
	select {}
}
