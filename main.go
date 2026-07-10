package main

import (
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"
)

// sendWoL sends a Wake-on-LAN magic packet for the given MAC to a UDP address.
func sendWoL(mac, dst string) error {
	hw, err := net.ParseMAC(mac)
	if err != nil {
		return err
	}
	packet := make([]byte, 0, 102)
	for i := 0; i < 6; i++ {
		packet = append(packet, 0xFF)
	}
	for i := 0; i < 16; i++ {
		packet = append(packet, hw...)
	}
	conn, err := net.Dial("udp", dst)
	if err != nil {
		return err
	}
	defer conn.Close()
	_, err = conn.Write(packet)
	return err
}

// reachable does a quick TCP dial to check whether the gamer's proxy answers.
func reachable(addr string, timeout time.Duration) bool {
	c, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return false
	}
	c.Close()
	return true
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func main() {
	mac := getenv("GAMER_MAC", "50:eb:f6:1f:93:59")
	target := getenv("GAMER_URL", "http://192.168.1.115:8080")
	tcpAddr := getenv("GAMER_TCP", "192.168.1.115:8080")
	broadcast := getenv("BROADCAST", "192.168.1.255:9")
	listen := getenv("LISTEN", ":8088")
	wakeTimeout := 45 * time.Second
	settle := 2 * time.Second

	u, err := url.Parse(target)
	if err != nil {
		log.Fatalf("bad GAMER_URL %q: %v", target, err)
	}
	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.FlushInterval = -1 // stream chunks immediately (needed for SSE / streaming chat)

	handler := func(w http.ResponseWriter, r *http.Request) {
		if !reachable(tcpAddr, time.Second) {
			log.Printf("%s %s -> gamer asleep, sending WoL to %s", r.Method, r.URL.Path, mac)
			_ = sendWoL(mac, broadcast)
			_ = sendWoL(mac, "255.255.255.255:9")
			deadline := time.Now().Add(wakeTimeout)
			for time.Now().Before(deadline) {
				if reachable(tcpAddr, time.Second) {
					break
				}
				_ = sendWoL(mac, broadcast)
				time.Sleep(2 * time.Second)
			}
			if !reachable(tcpAddr, time.Second) {
				http.Error(w, "gamer did not wake in time", http.StatusGatewayTimeout)
				log.Printf("gamer did not wake within %s", wakeTimeout)
				return
			}
			log.Printf("gamer awake, forwarding")
			time.Sleep(settle) // let Ollama finish coming up
		}
		proxy.ServeHTTP(w, r)
	}

	log.Printf("WoL-Ollama proxy listening on %s -> %s (mac %s, broadcast %s)", listen, target, mac, broadcast)
	log.Fatal(http.ListenAndServe(listen, http.HandlerFunc(handler)))
}
