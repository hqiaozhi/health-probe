package tools

import (
	"health-probe/internal/core/tools/dns"
	"health-probe/internal/core/tools/http"
	"health-probe/internal/core/tools/ping"
	"health-probe/internal/core/tools/tcp"
)

type Tools struct {
	HTTP http.HTTP
	TCP  tcp.TCP
	DNS  dns.DNS
	PING ping.PING
}
