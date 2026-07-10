package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"syscall"
	"time"
)

type options struct {
	GamerMac   string `json:"gamer_mac"`
	GamerURL   string `json:"gamer_url"`
	GamerTCP   string `json:"gamer_tcp"`
	Broadcast  string `json:"broadcast"`
	ListenPort int    `json:"listen_port"`
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

// sendWoL sends a Wake-on-LAN magic packet. SO_BROADCAST must be set or the
// kernel rejects broadcast sends (EACCES) on Linux.
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
	_, err = conn.Write(packet)
	return err
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
	if err := sendWoL(mac, broadcast); err != nil {
		log.Printf("WoL error to %s: %v", broadcast, err)
	}
	if err := sendWoL(mac, "255.255.255.255:9"); err != nil {
		log.Printf("WoL error to 255.255.255.255:9: %v", err)
	}
}

func main() {
	o := loadOptions()
	mac := pick(o.GamerMac, os.Getenv("GAMER_MAC"), "50:eb:f6:1f:93:59")
	target := pick(o.GamerURL, os.Getenv("GAMER_URL"), "http://192.168.1.115:8080")
	tcpAddr := pick(o.GamerTCP, os.Getenv("GAMER_TCP"), "192.168.1.115:8080")
	broadcast := pick(o.Broadcast, os.Getenv("BROADCAST"), "192.168.1.255:9")
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
		log.Fatalf("bad gamer_url %q: %v", target, err)
	}
	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.FlushInterval = -1

	handler := func(w http.ResponseWriter, r *http.Request) {
		if !reachable(tcpAddr, time.Second) {
			log.Printf("%s %s -> gamer asleep; sending WoL to %s (broadcast %s)", r.Method, r.URL.Path, mac, broadcast)
			wake(mac, broadcast)
			deadline := time.Now().Add(wakeTimeout)
			for time.Now().Before(deadline) {
				if reachable(tcpAddr, time.Second) {
					break
				}
				wake(mac, broadcast)
				time.Sleep(2 * time.Second)
			}
			if !reachable(tcpAddr, time.Second) {
				http.Error(w, "gamer did not wake in time", http.StatusGatewayTimeout)
				log.Printf("gamer did not wake within %s", wakeTimeout)
				return
			}
			log.Printf("gamer awake, forwarding")
			time.Sleep(settle)
		}
		proxy.ServeHTTP(w, r)
	}

	log.Printf("WoL-Ollama proxy listening on %s -> %s (mac %s, broadcast %s)", listen, target, mac, broadcast)
	log.Fatal(http.ListenAndServe(listen, http.HandlerFunc(handler)))
}
