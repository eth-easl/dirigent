package proxy

import (
	"net/http"
	"strings"
)

func consumeNode(xff string, from int) (string, int) {
	if xff == "" {
		return "", 0
	}
	// The x-forwarded-header consists of multiple nodes, split by ","
	rest := xff[from:]
	i := strings.Index(rest, ",")
	if i == -1 {
		i = len(rest)
	}
	return strings.TrimSpace(rest[:i]), from + i + 1
}

func writeNode(fwd *strings.Builder, node string) {
	// For simplicity, an address is IPv6 it contains a colon (:)
	ipv6 := strings.Contains(node, ":")
	if ipv6 {
		// Convert IPv6 address to "[ipv6 addr]" format
		fwd.WriteString(`"[`)
	}
	fwd.WriteString(node)
	if ipv6 {
		fwd.WriteString(`]"`)
	}
}

func generateForwarded(xff, xfp, xfh string) string {
	fwd := &strings.Builder{}
	// The size is dominated by the side of the individual headers.
	// + 5 + 1 for host= and delimiter
	// + 6 + 1 for proto= and delimiter
	// + (5 + 4) * x for each for= clause and delimiter (assuming ipv6)
	fwd.Grow(len(xff) + len(xfp) + len(xfh) + 6 + 7 + 9*(strings.Count(xff, ",")+1))

	node, next := consumeNode(xff, 0)
	if xff != "" {
		fwd.WriteString("for=")
		writeNode(fwd, node)
	}
	if xfh != "" {
		if node != "" {
			fwd.WriteRune(';')
		}
		fwd.WriteString("host=")
		fwd.WriteString(xfh)
	}
	if xfp != "" {
		if node != "" || xfh != "" {
			fwd.WriteRune(';')
		}
		fwd.WriteString("proto=")
		fwd.WriteString(xfp)
	}

	for next < len(xff) {
		node, next = consumeNode(xff, next)
		fwd.WriteString(", for=")
		writeNode(fwd, node)
	}

	return fwd.String()
}

func ForwardedShimHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer h.ServeHTTP(w, r)

		// Forwarded: by=<identifier>;for=<identifier>;host=<host>;proto=<http|https>
		fwd := r.Header.Get("Forwarded")

		// Don't add a shim if the header is already present
		if fwd != "" {
			return
		}

		// X-Forwarded-For: <client>, <proxy1>, <proxy2>
		xff := r.Header.Get("X-Forwarded-For")
		// X-Forwarded-Proto: <protocol>
		xfp := r.Header.Get("X-Forwarded-Proto")
		// X-Forwarded-Host: <host>
		xfh := r.Header.Get("X-Forwarded-Host")

		// Nothing to do if we don't have any X-Forwarded-* headers
		if xff == "" && xfp == "" && xfh == "" {
			return
		}

		r.Header.Set("Forwarded", generateForwarded(xff, xfp, xfh))
	})
}
