package discovery

import (
	"context"
	"crypto/tls"
	"html"
	"io"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var titlePattern = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
var tagPattern = regexp.MustCompile(`(?s)<[^>]*>`)

func isWebPort(port int) bool {
	switch port {
	case 80, 443, 3000, 5000, 5001, 5055, 6767, 7878, 8000, 8006, 8008,
		8080, 8081, 8090, 8096, 8123, 8384, 8443, 8686, 8888, 8989, 9000,
		9090, 9443, 9696, 32400:
		return true
	}
	return false
}

func isTLSPort(port int) bool {
	switch port {
	case 443, 8006, 8443, 8883, 9443:
		return true
	}
	return false
}

func (s *Scanner) identify(ctx context.Context, ip string, port int) (product, source string) {
	if port == 22 {
		if banner := sshBanner(ip, s.timeout); banner != "" {
			return banner, "ssh"
		}
	}
	if isWebPort(port) {
		if title := s.httpTitle(ctx, ip, port); title != "" {
			return title, "http"
		}
	}
	if isTLSPort(port) {
		if name := tlsName(ip, port, s.timeout); name != "" {
			return name, "tls"
		}
	}
	return "", "tcp"
}

func (s *Scanner) httpTitle(ctx context.Context, ip string, port int) string {
	scheme := "http"
	if isTLSPort(port) {
		scheme = "https"
	}
	url := scheme + "://" + net.JoinHostPort(ip, strconv.Itoa(port)) + "/"
	reqCtx, cancel := context.WithTimeout(ctx, 1800*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", "hearthlight/0.1 (home network map)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	resp, err := s.http.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 96<<10))
	if err != nil {
		return ""
	}
	match := titlePattern.FindSubmatch(body)
	if len(match) < 2 {
		return ""
	}
	title := cleanTitle(string(match[1]))
	if title == "" || genericTitle(title) {
		return ""
	}
	return title
}

func cleanTitle(raw string) string {
	raw = tagPattern.ReplaceAllString(raw, " ")
	raw = html.UnescapeString(raw)
	raw = strings.Join(strings.Fields(raw), " ")
	runes := []rune(raw)
	if len(runes) > 80 {
		raw = string(runes[:80])
	}
	return raw
}

func genericTitle(title string) bool {
	t := strings.ToLower(strings.TrimSpace(title))
	if t == "" {
		return true
	}
	for _, generic := range []string{
		"welcome to nginx", "apache2 default", "it works!", "404 not found",
		"403 forbidden", "401 authorization", "401 unauthorized",
		"default page", "index of /", "document moved", "redirecting",
		"site not found", "no such site",
	} {
		if strings.Contains(t, generic) {
			return true
		}
	}
	return false
}

func sshBanner(ip string, timeout time.Duration) string {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, "22"), timeout)
	if err != nil {
		return ""
	}
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(1200 * time.Millisecond))
	buf := make([]byte, 256)
	n, err := conn.Read(buf)
	if err != nil || n == 0 {
		return ""
	}
	line := strings.SplitN(string(buf[:n]), "\n", 2)[0]
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "SSH-2.0-")
	line = strings.TrimPrefix(line, "SSH-1.99-")
	if len(line) > 64 {
		line = line[:64]
	}
	return line
}

func tlsName(ip string, port int, timeout time.Duration) string {
	dialer := &net.Dialer{Timeout: timeout}
	// Homelab services routinely use self-signed certificates. We only read the
	// certificate name for identification and never use this connection for
	// anything security-sensitive.
	conn, err := tls.DialWithDialer(dialer, "tcp", net.JoinHostPort(ip, strconv.Itoa(port)), &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		return ""
	}
	defer conn.Close()
	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return ""
	}
	name := strings.TrimSpace(certs[0].Subject.CommonName)
	if name == "" && len(certs[0].DNSNames) > 0 {
		name = strings.TrimSpace(certs[0].DNSNames[0])
	}
	if name == "" || strings.EqualFold(name, "localhost") {
		return ""
	}
	runes := []rune(name)
	if len(runes) > 64 {
		name = string(runes[:64])
	}
	return name
}
