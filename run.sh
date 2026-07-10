#!/usr/bin/with-contenv bashio
export GAMER_MAC="$(bashio::config 'gamer_mac')"
export GAMER_URL="$(bashio::config 'gamer_url')"
export GAMER_TCP="$(bashio::config 'gamer_tcp')"
export BROADCAST="$(bashio::config 'broadcast')"
export LISTEN=":$(bashio::config 'listen_port')"
bashio::log.info "WoL Ollama Proxy starter (mac ${GAMER_MAC}, mål ${GAMER_URL}, port ${LISTEN})"
exec /usr/bin/wolproxy
