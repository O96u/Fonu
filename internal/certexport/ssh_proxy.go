package certexport

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	certstore "github.com/fonu/fonu/internal/certificate"
	"golang.org/x/crypto/ssh"
	"golang.org/x/net/proxy"
)

func sshClientConn(cfg certstore.SSHExportConfig) (*ssh.Client, error) {
	auth, err := sshAuthMethods(cfg)
	if err != nil {
		return nil, err
	}
	port := cfg.Port
	if port <= 0 {
		port = 22
	}
	clientConfig := &ssh.ClientConfig{
		User:            cfg.Username,
		Auth:            auth,
		HostKeyCallback: func(_ string, _ net.Addr, key ssh.PublicKey) error {
			actual := ssh.FingerprintSHA256(key)
			if cfg.HostKeyFingerprint == "" { return fmt.Errorf("尚未配置可信 SSH 主机指纹；请在服务器端核实 SHA256 指纹后填写") }
			if actual != cfg.HostKeyFingerprint { return fmt.Errorf("SSH 主机指纹不匹配，已阻止连接") }
			return nil
		},
		Timeout:         15 * time.Second,
	}
	target := net.JoinHostPort(strings.Trim(cfg.Host, "[]"), strconv.Itoa(port))
	conn, err := dialTCP(cfg, target)
	if err != nil {
		return nil, err
	}
	_ = conn.SetDeadline(time.Now().Add(30*time.Second))
	cc, chans, reqs, err := ssh.NewClientConn(conn, target, clientConfig)
	if err != nil {
		conn.Close()
		return nil, err
	}
	_ = conn.SetDeadline(time.Time{})
	return ssh.NewClient(cc, chans, reqs), nil
}

func dialTCP(cfg certstore.SSHExportConfig, targetAddr string) (net.Conn, error) {
	if !cfg.ProxyEnabled {
		return net.DialTimeout("tcp", targetAddr, 15*time.Second)
	}
	proxyPort := cfg.ProxyPort
	if proxyPort <= 0 {
		if cfg.ProxyType == certstore.SSHProxyHTTP {
			proxyPort = 8080
		} else {
			proxyPort = 1080
		}
	}
	proxyAddr := net.JoinHostPort(strings.Trim(cfg.ProxyHost, "[]"), strconv.Itoa(proxyPort))
	switch cfg.ProxyType {
	case certstore.SSHProxyHTTP:
		return dialHTTPProxy(proxyAddr, targetAddr, cfg.ProxyUsername, cfg.ProxyPassword)
	default:
		var auth *proxy.Auth
		if cfg.ProxyUsername != "" || cfg.ProxyPassword != "" {
			auth = &proxy.Auth{User: cfg.ProxyUsername, Password: cfg.ProxyPassword}
		}
		dialer, err := proxy.SOCKS5("tcp", proxyAddr, auth, &net.Dialer{Timeout: 15 * time.Second})
		if err != nil {
			return nil, fmt.Errorf("SOCKS5 代理：%w", err)
		}
		return dialer.Dial("tcp", targetAddr)
	}
}

func dialHTTPProxy(proxyAddr, targetAddr, user, pass string) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", proxyAddr, 15*time.Second)
	if err != nil {
		return nil, fmt.Errorf("连接 HTTP 代理：%w", err)
	}
	_ = conn.SetDeadline(time.Now().Add(15*time.Second))
	var reqBuilder strings.Builder
	reqBuilder.WriteString(fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n", targetAddr, targetAddr))
	if user != "" || pass != "" {
		token := base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
		reqBuilder.WriteString("Proxy-Authorization: Basic " + token + "\r\n")
	}
	reqBuilder.WriteString("\r\n")
	if _, err := conn.Write([]byte(reqBuilder.String())); err != nil {
		conn.Close()
		return nil, err
	}
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, &http.Request{Method: http.MethodConnect})
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("HTTP 代理响应：%w", err)
	}
	if resp.Body != nil {
		resp.Body.Close()
	}
	if resp.StatusCode != http.StatusOK {
		conn.Close()
		return nil, fmt.Errorf("HTTP 代理 CONNECT 失败：%s", strings.TrimSpace(resp.Status))
	}
	_ = conn.SetDeadline(time.Time{})
	return conn, nil
}
