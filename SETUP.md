# AI-setup — samlet guide (WOL-proxy + sidste småting)

Denne guide samler det hele: udrulning af **WOL-proxyen** (så Copilot vækker gameren automatisk), oprettelse af de to **HASS.Agent-kommandoer** (stop/start Ollama), samt de sidste **småting** (keep-alive og oprydning).

## Arkitektur

```
GitHub Copilot (Mac)
      │  http://<HA-IP>:8088/v1
      ▼
WOL-proxy  (denne add-on, kører altid på HA)
      │  1) svarer gameren?  nej → send WoL, vent til vågen
      │  2) reverse-proxy →
      ▼
ollama-metrics proxy  (gamer:8080)   ← tæller tokens
      ▼
Ollama  (gamer:11434)
```

Du beholder S3-sleep + "kun magic packet" på gameren (ingen spontane opvågninger), men Copilot kan sende en prompt uden at du manuelt vækker maskinen.

---

## Del 1 — WOL-proxyen (HA add-on)

Add-on'en skal køre på en **altid-tændt** maskine. Din HA er det oplagte valg (Unraid-toweren er kun kortvarigt tændt).

### 1.1 Hent koden
```bash
git clone https://github.com/bondesen/wol-ollama-proxy.git
```
(privat repo — kræver login/PAT).

### 1.2 Læg add-on'en på HA
Kopiér repo-filerne til `/addons/wol_ollama_proxy/` på HA-maskinen. Nemmeste veje:
- **Samba share**-add-on → `\\<HA-IP>\addons` → opret mappen `wol_ollama_proxy` og læg filerne, eller
- **Studio Code Server / File editor**-add-on → opret mappen + filer, eller
- SSH.

Disse filer skal ligge direkte i mappen: `config.yaml`, `build.yaml`, `Dockerfile`, `run.sh`, `main.go`, `go.mod`.

### 1.3 Installér
Indstillinger → Add-ons → Add-on Store → menuen ⋮ → **Check for updates** → scroll til **"Local add-ons"** → **WoL Ollama Proxy** → **Install** (bygger imaget, tager et par min).

### 1.4 Konfigurér (fanen Configuration)
```yaml
gamer_mac: "50:eb:f6:1f:93:59"
gamer_url: "http://192.168.1.115:8080"
gamer_tcp: "192.168.1.115:8080"
broadcast: "192.168.1.255:9"
listen_port: 8088
```
Ret `broadcast` hvis dit subnet ikke er `192.168.1.x`.

### 1.5 Start
Fanen Info → slå **Start on boot** + **Watchdog** til → **Start**. Tjek **Log** — den skal skrive at proxyen lytter.

### 1.6 Peg Copilot på HA
Skift Ollama-endpointet i Copilot fra gameren til HA:
```
http://<HA-IP>:8088/v1
```
(behold `/v1`).

### (Valgfrit) Byg binæren standalone med Go
Du har Go — vil du teste binæren direkte (fx på en Linux-boks der er altid tændt):
```bash
cd wol-ollama-proxy
go build -o wolproxy .
GAMER_MAC=50:eb:f6:1f:93:59 GAMER_URL=http://192.168.1.115:8080 GAMER_TCP=192.168.1.115:8080 BROADCAST=192.168.1.255:9 LISTEN=:8088 ./wolproxy
```
> Bemærk: gameren selv duer ikke som host — den sover, og så kan proxyen ikke modtage prompten der vækker den. Den skal køre et altid-tændt sted (HA).

---

## Del 2 — HASS.Agent-kommandoer (stop/start Ollama)

Så kan du styre selve `ollama serve`-processen fra dashboardet. Kør HASS.Agent **som administrator** (schtasks på en SYSTEM-opgave kræver det).

### 2.1 Opret "Start Ollama"
HASS.Agent → **Commands** → **Add** →
- **Type:** Custom command
- **Name:** `Start Ollama`
- **Command:** `schtasks /run /tn "OllamaServe"`
- Sæt flueben i at den udstilles som `button` i HA.

### 2.2 Opret "Stop Ollama"
Samme fremgangsmåde →
- **Name:** `Stop Ollama`
- **Command:** `schtasks /end /tn "OllamaServe"`

### 2.3 Sig til mig
Når de to kommandoer er oprettet, dukker de op i HA som `button.gamer_*`-entiteter. **Send mig deres entity-id'er**, så wirer jeg **Start/Stop**-knapper ind i Ollama-styring-sektionen på dashboardet.

> Alternativ uden admin: vil du bare frigøre VRAM (ikke stoppe processen), så er dashboardets **Unload**-knap nok — den bruger `keep_alive: 0` via API'et.

---

## Del 3 — Ollama keep-alive (stop hurtig unloading)

Ollama smider modeller ud af VRAM efter **5 min** som standard. Vil du beholde dem indlæst, tilføj en linje i `C:\Users\bonde\start-ollama.bat`:

```bat
@echo off
set OLLAMA_MODELS=C:\Users\bonde\.ollama\models
set OLLAMA_HOST=127.0.0.1:11434
set OLLAMA_KEEP_ALIVE=-1
"C:\Users\bonde\AppData\Local\Programs\Ollama\ollama.exe" serve
```

`-1` = behold for evigt (mens maskinen er vågen), eller fx `30m` / `1h`. Genstart tjenesten:
```powershell
schtasks /end /tn "OllamaServe"
schtasks /run /tn "OllamaServe"
```

---

## Del 4 — Oprydning: fjern ubrugt shell_command

Nu hvor Ollama-styringen bruger `rest_command` (fra `/config/packages/ollama.yaml`), er de gamle `shell_command`-linjer overflødige. Fjern disse fire fra `shell_command:`-blokken i `configuration.yaml`:
`ollama_unload`, `ollama_load`, `ollama_pull`, `ollama_delete`.
(Behold `icloud_slideshow_fetch`.) Genstart HA bagefter.

---

## Verifikation (den rigtige test)

1. Lad gameren gå i **sleep**.
2. Send en prompt fra **Copilot**.
3. I add-on-loggen: `sending WoL` → `gamer awake, forwarding`.
4. Copilot svarer (første prompt langsom pga. vækning, resten hurtige).
5. Dashboardet: `Ollama online` = til, tokens tikker op.

Virker det → hele kæden er selvkørende: Copilot vækker gameren, får svar, og alt tælles.
