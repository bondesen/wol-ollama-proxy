# WoL Ollama Proxy — lokal Home Assistant add-on

> 📘 **Komplet setup (add-on + HASS.Agent-kommandoer + keep-alive + oprydning): se [SETUP.md](SETUP.md).**

Vækker gaming-PC'en automatisk når der kommer en Ollama-forespørgsel, og reverse-proxyer videre til gamerens metrics-proxy. Så beholder du S3-sleep + "kun magic packet" (ingen spontane opvågninger), men Copilot kan sende en prompt uden at du manuelt vækker maskinen.

**Kæde:** Copilot → `HA:8088` (denne add-on) → `gamer:8080` (metrics-proxy) → Ollama `11434`. Tokens tælles stadig.

## Filer
`config.yaml`, `build.yaml`, `Dockerfile`, `run.sh`, `main.go`, `go.mod`

## Hurtig installation (HA add-on)
1. Kopiér repoet til `/addons/wol_ollama_proxy/` på HA-maskinen (Samba / Studio Code Server / SSH).
2. Indstillinger → Add-ons → Add-on Store → ⋮ → **Check for updates** → **Local add-ons** → **WoL Ollama Proxy** → **Install**.
3. Fanen **Configuration**: tjek `gamer_mac`, `gamer_url`, `gamer_tcp`, `broadcast`, `listen_port`.
4. **Start** + slå *Start on boot* og *Watchdog* til.
5. Peg Copilot på `http://<HA-IP>:8088/v1`.

Se **[SETUP.md](SETUP.md)** for hele opsætningen inkl. HASS.Agent-kommandoer, keep-alive og oprydning.

## Byg standalone med Go
```bash
go build -o wolproxy .
GAMER_MAC=50:eb:f6:1f:93:59 GAMER_URL=http://192.168.1.115:8080 GAMER_TCP=192.168.1.115:8080 BROADCAST=192.168.1.255:9 LISTEN=:8088 ./wolproxy
```

## Miljøvariabler
| Variabel | Standard | Beskrivelse |
|---|---|---|
| `GAMER_MAC` | `50:eb:f6:1f:93:59` | Gamerens MAC til WoL |
| `GAMER_URL` | `http://192.168.1.115:8080` | Reverse-proxy mål (metrics-proxy) |
| `GAMER_TCP` | `192.168.1.115:8080` | TCP-adresse der pinges for at se om gameren er vågen |
| `BROADCAST` | `192.168.1.255:9` | UDP broadcast til magic packet |
| `LISTEN` | `:8088` | Lytteport |

## Fejlfinding
- **Vågner ikke:** tjek at `BROADCAST` matcher dit subnet, og at NIC'ens WoL ("kun magic packet") er slået til på gameren.
- **504:** gameren svarede ikke i tide — se loggen; øg evt. `wakeTimeout` i `main.go`.
- **Streaming hænger:** add-on'en flusher løbende (`FlushInterval = -1`), og gamerens patchede metrics-proxy streamer `/v1/chat/completions`.
