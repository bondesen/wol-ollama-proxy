# Node 22. Debian bookworm leverer node 20 (EOL), og stdio-MCP-servere
# (fx npx @mem0/mcp-server) kraever 22. Vi kopierer binaeren ind.
FROM node:22-bookworm-slim AS node_source

ARG BUILD_FROM
FROM ${BUILD_FROM}

SHELL ["/bin/bash", "-o", "pipefail", "-c"]

# HERMES_HOME -> /data = add-on'ens persistente volumen (overlever opdatering).
# Nous' image bruger /opt/data; vi omdirigerer alt til /data.
ENV HERMES_HOME=/data \
    HERMES_WRITE_SAFE_ROOT=/data \
    HERMES_LAZY_INSTALL_TARGET=/data/lazy-packages \
    HERMES_GATEWAY_BOOTSTRAP_STATE=running \
    PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1 \
    UV_LINK_MODE=copy \
    PATH="/opt/hermes/.venv/bin:/usr/local/bin:${PATH}"

# Systempakker - kun det magre saet.
#   ripgrep       : agentens fil-soegning shell'er ud til `rg`
#   openssh-client: SSH-backenden hard-checker `ssh` og `scp` via shutil.which
#   git           : skills-hub, plugin-discovery, agentens git-operationer
# Udeladt bevidst: ffmpeg (TTS), libolm (Matrix), docker-cli, gcc/cmake.
RUN apt-get update && apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        git \
        ripgrep \
        openssh-client \
        procps \
        tar \
    && rm -rf /var/lib/apt/lists/*

COPY --from=node_source /usr/local/bin/node /usr/local/bin/node
COPY --from=node_source /usr/local/lib/node_modules/npm /usr/local/lib/node_modules/npm
RUN ln -sf /usr/local/lib/node_modules/npm/bin/npm-cli.js /usr/local/bin/npm \
    && ln -sf /usr/local/lib/node_modules/npm/bin/npx-cli.js /usr/local/bin/npx \
    && node --version && npm --version

RUN curl -LsSf https://astral.sh/uv/install.sh | env UV_INSTALL_DIR=/usr/local/bin sh \
    && uv --version

ARG HERMES_REF=main
RUN git clone --depth 1 --branch "${HERMES_REF}" \
        https://github.com/NousResearch/hermes-agent.git /opt/hermes

WORKDIR /opt/hermes

# Lean install. Bevidst UDELADT:
#   voice   -> faster-whisper + onnxruntime (hundredvis af MB)
#   matrix  -> python-olm, kraever libolm-dev + gcc/cmake
#   messaging -> traekker discord[voice] + slack med; vi vil kun have telegram
#   modal/daytona -> alternative terminal-backends vi ikke bruger
# Chromium kommer IKKE fra Python, men fra `npx playwright install` - som vi
# simpelthen ikke koerer.
RUN uv venv --python 3.13 /opt/hermes/.venv \
    && VIRTUAL_ENV=/opt/hermes/.venv uv pip install --no-cache-dir \
         -e ".[mcp,cli,homeassistant]" \
    && VIRTUAL_ENV=/opt/hermes/.venv uv pip install --no-cache-dir \
         "python-telegram-bot[webhooks]==22.6" \
    && /opt/hermes/.venv/bin/hermes --version || true

COPY rootfs /

RUN chmod a+x /etc/s6-overlay/s6-rc.d/init-hermes/run \
    && chmod a+x /etc/s6-overlay/s6-rc.d/gateway-default/run \
    && chmod a+x /etc/s6-overlay/s6-rc.d/gateway-default/finish

LABEL io.hass.version="0.1.0" io.hass.type="addon" io.hass.arch="amd64"
