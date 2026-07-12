package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"syscall"
	"time"
)

type options struct {
	GamerMac   string `json:"gamer_mac"`
	GamerURL   string `json:"gamer_url"`
	GamerTCP   string `json:"gamer_tcp"`
	Broadcast  string `json:"broadcast"`
	ListenPort int    `json:"listen_port"`
	Debug      bool   `json:"debug"`
}

func loadOptions() options {
	var o options
	if data, err := os.ReadFile("/data/options.json"); err == nil {
		_ = json.Unmarshal(data, &o)
	}
	return o
}

func pick(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// sendWoL sender en Wake-on-LAN magic packet. SO_BROADCAST SKAL saettes,
// ellers afviser kernen broadcast-sends paa Linux.
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
	raddr, err := net.ResolveUDPAddr("udp4", dst)
	if err != nil {
		return err
	}
	conn, err := net.DialUDP("udp4", nil, raddr)
	if err != nil {
		return err
	}
	defer conn.Close()
	if rc, e := conn.SyscallConn(); e == nil {
		_ = rc.Control(func(fd uintptr) {
			_ = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
		})
	}
	n, err := conn.Write(packet)
	if err != nil {
		return err
	}
	log.Printf("WoL sendt -> %s (%d bytes, mac %s)", dst, n, mac)
	return nil
}

func reachable(addr string, timeout time.Duration) bool {
	c, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return false
	}
	c.Close()
	return true
}

func wake(mac, broadcast string) {
	for _, dst := range []string{broadcast, "255.255.255.255:9"} {
		if err := sendWoL(mac, dst); err != nil {
			log.Printf("WoL FEJL -> %s: %v", dst, err)
		}
	}
}

// inspect logger hvad klienten FAKTISK sender: body-stoerrelse, model, stream-flag
// og hvor mange tools der er vedhaeftet. Body'en afspilles igen saa proxyen kan
// videresende den uaendret.
func inspect(r *http.Request) {
	if r.Body == nil {
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("DEBUG: kunne ikke laese body: %v", err)
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	r.ContentLength = int64(len(body))
	if len(body) == 0 {
		return
	}

	var m map[string]interface{}
	if err := json.Unmarshal(body, &m); err != nil {
		log.Printf("DEBUG: %s %s | %d bytes | ikke-JSON body", r.Method, r.URL.Path, len(body))
		return
	}
	tools := 0
	var names []string
	if t, ok := m["tools"].([]interface{}); ok {
		tools = len(t)
		for _, it := range t {
			if tm, ok := it.(map[string]interface{}); ok {
				if fn, ok := tm["function"].(map[string]interface{}); ok {
					if n, ok := fn["name"].(string); ok {
						names = append(names, n)
					}
				}
			}
		}
	}
	msgs := 0
	if ms, ok := m["messages"].([]interface{}); ok {
		msgs = len(ms)
	}
	log.Printf("DEBUG: %s %s | %d bytes | model=%v | stream=%v | messages=%d | TOOLS=%d",
		r.Method, r.URL.Path, len(body), m["model"], m["stream"], msgs, tools)
	if tools > 0 {
		limit := len(names)
		suffix := ""
		if limit > 20 {
			limit = 20
			suffix = " ..."
		}
		log.Printf("DEBUG: tools: %s%s", strings.Join(names[:limit], ", "), suffix)
	}
}

func main() {
	o := loadOptions()
	mac := pick(o.GamerMac, os.Getenv("GAMER_MAC"), "50:eb:f6:1f:93:59")
	target := pick(o.GamerURL, os.Getenv("GAMER_URL"), "http://192.168.1.115:8080")
	tcpAddr := pick(o.GamerTCP, os.Getenv("GAMER_TCP"), "192.168.1.115:8080")
	broadcast := pick(o.Broadcast, os.Getenv("BROADCAST"), "192.168.1.255:9")
	debug := o.Debug || os.Getenv("DEBUG") == "true"
	listen := os.Getenv("LISTEN")
	if o.ListenPort != 0 {
		listen = fmt.Sprintf(":%d", o.ListenPort)
	}
	if listen == "" {
		listen = ":8088"
	}
	wakeTimeout := 60 * time.Second
	settle := 2 * time.Second

	u, err := url.Parse(target)
	if err != nil {
		log.Fatalf("ugyldig gamer_url %q: %v", target, err)
	}
	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.FlushInterval = -1

	handler := func(w http.ResponseWriter, r *http.Request) {
		if debug {
			inspect(r)
		}
		if reachable(tcpAddr, time.Second) {
			log.Printf("%s %s -> gamer vaagen, forwarder", r.Method, r.URL.Path)
			proxy.ServeHTTP(w, r)
			return
		}

		log.Printf("%s %s -> gamer SOVER, vaekker (mac %s, broadcast %s)", r.Method, r.URL.Path, mac, broadcast)
		start := time.Now()
		wake(mac, broadcast)
		deadline := start.Add(wakeTimeout)
		for time.Now().Before(deadline) {
			if reachable(tcpAddr, time.Second) {
				break
			}
			time.Sleep(2 * time.Second)
			if !reachable(tcpAddr, time.Second) {
				wake(mac, broadcast)
			}
		}
		if !reachable(tcpAddr, time.Second) {
			log.Printf("TIMEOUT: gamer vaagnede ikke inden for %s", wakeTimeout)
			http.Error(w, "gamer did not wake in time", http.StatusGatewayTimeout)
			return
		}
		log.Printf("gamer VAAGEN efter %.0fs, forwarder", time.Since(start).Seconds())
		time.Sleep(settle)
		proxy.ServeHTTP(w, r)
	}

	log.Printf("WoL-Ollama proxy lytter paa %s -> %s (mac %s, broadcast %s, debug=%v)", listen, target, mac, broadcast, debug)
	log.Fatal(http.ListenAndServe(listen, http.HandlerFunc(handler)))
}
