# WoL Ollama Proxy — lokal Home Assistant app (add-on)

> 📘 **Komplet setup: se [SETUP.md](SETUP.md).**
>
> ℹ️ I Home Assistant 2026.2+ hedder "Add-ons" nu **"Apps"** — samme ting. Du **kompilerer ikke Go selv**; HA gør det automatisk ved **Install**.

Vækker gaming-PC'en automatisk når der kommer en Ollama-forespørgsel, og reverse-proxyer videre til gamerens metrics-proxy.

**Kæde:** Copilot → `HA:8088` (denne app) → `gamer:8080` (metrics-proxy) → Ollama `11434`.

## Filer
`config.yaml`, `build.yaml`, `Dockerfile`, `run.sh`, `main.go`, `go.mod`

## Hurtig installation (HA app)
1. Kopiér til `/addons/wol_ollama_proxy/` (`config.yaml` i roden).
2. Indstillinger → **Apps** → **App Store** → ⋮ → **Check for updates** → F5 → **Local apps** → **WoL Ollama Proxy** → **Install** (HA kompilerer Go automatisk).
3. Konfigurér `gamer_mac` osv. → **Start** + *Start on boot* + *Watchdog*.
4. Peg Copilot på `http://<HA-IP>:8088/v1`.

## Miljøvariabler
| Variabel | Standard |
|---|---|
| `GAMER_MAC` | `50:eb:f6:1f:93:59` |
| `GAMER_URL` | `http://192.168.1.115:8080` |
| `GAMER_TCP` | `192.168.1.115:8080` |
| `BROADCAST` | `192.168.1.255:9` |
| `LISTEN` | `:8088` |

## Fejlfinding
- **Appen dukker ikke op:** `config.yaml` skal ligge i mappens rod under `/addons/...`; kør Check for updates + F5.
- **Vågner ikke:** tjek `BROADCAST` matcher subnettet + NIC-WoL.
