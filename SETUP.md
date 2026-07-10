# AI-setup — samlet guide (WOL-proxy + sidste småting)

Denne guide samler det hele: udrulning af **WOL-proxyen** (så Copilot vækker gameren automatisk), oprettelse af de to **HASS.Agent-kommandoer** (stop/start Ollama), samt de sidste **småting** (keep-alive og oprydning).

> ⚠️ **"Add-ons" hedder nu "Apps"** i Home Assistant 2026.2+ (feb 2026). Funktionelt det samme — stadig Docker-containere via Supervisor.
>
> 🛠️ **Du skal IKKE kompilere Go selv.** HA bygger imaget og kompilerer `main.go` automatisk når du trykker **Install** (to-trins Dockerfile). Du lægger kun kildekoden ind.

## Arkitektur

```
GitHub Copilot (Mac)
      │  http://<HA-IP>:8088/v1
      ▼
WOL-proxy  (denne app, kører altid på HA)
      │  1) svarer gameren?  nej → send WoL, vent til vågen
      │  2) reverse-proxy →
      ▼
ollama-metrics proxy  (gamer:8080)   ← tæller tokens
      ▼
Ollama  (gamer:11434)
```

## Del 1 — WOL-proxyen (HA lokal app)

Appen skal køre på en **altid-tændt** maskine. Din HA er det oplagte valg.

### 1.1 Læg filerne på HA
Kopiér repo-filerne til **`/addons/wol_ollama_proxy/`** (mappen `/addons` er uændret — kun UI-navnet blev til "Apps"). `config.yaml` skal ligge i **mappens rod**. Filer: `config.yaml`, `build.yaml`, `Dockerfile`, `run.sh`, `main.go`, `go.mod`. Veje: Samba-app (`\\<HA-IP>\addons`), Studio Code Server / File editor, eller SSH.

### 1.2 Find appen
Indstillinger → **Apps** → **App Store** → ⋮ → **Check for updates** → **F5** → sektionen **"Local apps"** → **WoL Ollama Proxy**.

### 1.3 Installér (HER kompileres Go automatisk)
Klik appen → **Install**. HA bygger imaget: trin 1 kompilerer `main.go`, trin 2 er runtime. Et par min første gang — følg **Log**-fanen.

### 1.4 Konfigurér
```yaml
gamer_mac: "50:eb:f6:1f:93:59"
gamer_url: "http://192.168.1.115:8080"
gamer_tcp: "192.168.1.115:8080"
broadcast: "192.168.1.255:9"
listen_port: 8088
```

### 1.5 Start
Slå **Start on boot** + **Watchdog** til → **Start**. Tjek **Log**.

### 1.6 Peg Copilot på HA
```
http://<HA-IP>:8088/v1
```

---

## Del 2 — HASS.Agent-kommandoer (stop/start Ollama)

Kør HASS.Agent **som administrator**.

**Start Ollama:** HASS.Agent → Commands → Add → Custom command → Name `Start Ollama` → Command `schtasks /run /tn "OllamaServe"` → udstil som `button`.

**Stop Ollama:** samme → Name `Stop Ollama` → Command `schtasks /end /tn "OllamaServe"`.

Når de er lavet, dukker de op som `button.gamer_*` — send mig entity-id'erne, så wirer jeg Start/Stop-knapper på dashboardet.

> Alternativ uden admin: dashboardets **Unload**-knap frigør VRAM (via `keep_alive: 0`).

---

## Del 3 — Ollama keep-alive

Tilføj i `C:\Users\bonde\start-ollama.bat`:
```bat
@echo off
set OLLAMA_MODELS=C:\Users\bonde\.ollama\models
set OLLAMA_HOST=127.0.0.1:11434
set OLLAMA_KEEP_ALIVE=-1
"C:\Users\bonde\AppData\Local\Programs\Ollama\ollama.exe" serve
```
Genstart: `schtasks /end /tn "OllamaServe"` + `schtasks /run /tn "OllamaServe"`.

---

## Del 4 — Oprydning

Fjern de fire ubrugte fra `shell_command:` i `configuration.yaml`: `ollama_unload`, `ollama_load`, `ollama_pull`, `ollama_delete` (behold `icloud_slideshow_fetch`). Genstart HA.

---

## (Valgfrit) Byg standalone med Go

**Kun** hvis du vil køre proxyen uden for HA. Til HA-app-ruten er dette IKKE nødvendigt.
```bash
go build -o wolproxy .
GAMER_MAC=50:eb:f6:1f:93:59 GAMER_URL=http://192.168.1.115:8080 GAMER_TCP=192.168.1.115:8080 BROADCAST=192.168.1.255:9 LISTEN=:8088 ./wolproxy
```
> Gameren selv duer ikke som host — den sover.

---

## Verifikation

1. Lad gameren gå i **sleep**. 2. Send en Copilot-prompt. 3. App-log: `sending WoL` → `gamer awake, forwarding`. 4. Copilot svarer. 5. Dashboard: `Ollama online` = til.
