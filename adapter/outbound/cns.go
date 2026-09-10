package outbound

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"

	N "github.com/metacubex/mihomo/common/net"
	"github.com/metacubex/mihomo/component/dialer"
	"github.com/metacubex/mihomo/component/proxydialer"
	C "github.com/metacubex/mihomo/constant"
	TC "github.com/metacubex/mihomo/transport/cns"
)

const (
	defaultCnsKey  = "Meng"
	defaultCnsFlag = "httpUDP"
)

type Cns struct {
	*Base
	option *CnsOption
}

type CnsOption struct {
	BasicOption
	Name     string            `proxy:"name"`
	Server   string            `proxy:"server"`
	Port     int               `proxy:"port"`
	Key      string            `proxy:"key,omitempty"`
	Password string            `proxy:"password,omitempty"`
	Flag     string            `proxy:"flag,omitempty"`
	UDP      bool              `proxy:"udp,omitempty"`
	Headers  map[string]string `proxy:"headers,omitempty"`
}

// StreamConnContext implements C.ProxyAdapter.
func (cns *Cns) StreamConnContext(ctx context.Context, conn net.Conn, metadata *C.Metadata) (net.Conn, error) {
	conn, err := cns.shakeHand(ctx, conn, metadata, false)
	if err != nil {
		return nil, err
	}
	return TC.NewCnsConn(conn, cns.option.Password), nil
}

// DialContext implements C.ProxyAdapter.
func (cns *Cns) DialContext(ctx context.Context, metadata *C.Metadata) (C.Conn, error) {
	return cns.DialContextWithDialer(ctx, dialer.NewDialer(cns.DialOptions()...), metadata)
}

// DialContextWithDialer implements C.ProxyAdapter.
func (cns *Cns) DialContextWithDialer(ctx context.Context, cDialer C.Dialer, metadata *C.Metadata) (_ C.Conn, err error) {
	cDialer, err = cns.wrapDialer(cDialer)
	if err != nil {
		return nil, err
	}

	conn, err := cDialer.DialContext(ctx, "tcp", cns.addr)
	if err != nil {
		return nil, fmt.Errorf("%s connect error: %w", cns.addr, err)
	}
	defer func() {
		safeConnClose(conn, err)
	}()

	conn, err = cns.StreamConnContext(ctx, conn, metadata)
	if err != nil {
		return nil, err
	}
	return NewConn(conn, cns), nil
}

// ListenPacketContext implements C.ProxyAdapter.
func (cns *Cns) ListenPacketContext(ctx context.Context, metadata *C.Metadata) (C.PacketConn, error) {
	return cns.ListenPacketWithDialer(ctx, dialer.NewDialer(cns.DialOptions()...), metadata)
}

// ListenPacketWithDialer implements C.ProxyAdapter.
func (cns *Cns) ListenPacketWithDialer(ctx context.Context, cDialer C.Dialer, metadata *C.Metadata) (_ C.PacketConn, err error) {
	if !cns.option.UDP {
		return nil, C.ErrNotSupport
	}
	if err = cns.ResolveUDP(ctx, metadata); err != nil {
		return nil, err
	}
	cDialer, err = cns.wrapDialer(cDialer)
	if err != nil {
		return nil, err
	}

	conn, err := cDialer.DialContext(ctx, "tcp", cns.addr)
	if err != nil {
		return nil, fmt.Errorf("%s connect error: %w", cns.addr, err)
	}
	defer func() {
		safeConnClose(conn, err)
	}()

	conn, err = cns.shakeHand(ctx, conn, metadata, true)
	if err != nil {
		return nil, err
	}
	return newPacketConn(TC.NewUDPPacketConn(conn, cns.option.Password), cns), nil
}

// SupportWithDialer implements C.ProxyAdapter.
func (cns *Cns) SupportWithDialer() C.NetWork {
	if cns.option.UDP {
		return C.ALLNet
	}
	return C.TCP
}

// ProxyInfo implements C.ProxyAdapter.
func (cns *Cns) ProxyInfo() C.ProxyInfo {
	info := cns.Base.ProxyInfo()
	info.DialerProxy = cns.option.DialerProxy
	return info
}

func (cns *Cns) wrapDialer(cDialer C.Dialer) (C.Dialer, error) {
	if cns.option.DialerProxy == "" {
		return cDialer, nil
	}
	return proxydialer.NewByName(cns.option.DialerProxy, cDialer)
}

func (cns *Cns) shakeHand(ctx context.Context, conn net.Conn, metadata *C.Metadata, udp bool) (_ net.Conn, err error) {
	if ctx.Done() != nil {
		done := N.SetupContextForConn(ctx, conn)
		defer done(&err)
	}

	request, err := cns.requestHeader(metadata, udp)
	if err != nil {
		return nil, err
	}
	if err = writeCnsRequest(conn, request); err != nil {
		return nil, fmt.Errorf("write CNS handshake: %w", err)
	}

	bufferedConn := N.NewBufferedConn(conn)
	response, err := http.ReadResponse(bufferedConn.Reader(), &http.Request{Method: http.MethodConnect})
	if err != nil {
		return nil, fmt.Errorf("read CNS handshake: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CNS handshake rejected: %s", response.Status)
	}
	return bufferedConn, nil
}

func (cns *Cns) requestHeader(metadata *C.Metadata, udp bool) (string, error) {
	if metadata == nil || !metadata.Valid() || metadata.DstPort == 0 {
		return "", errors.New("invalid CNS destination metadata")
	}

	encodedTarget := TC.EncryptHost(metadata.RemoteAddress(), cns.option.Password)
	var builder strings.Builder
	builder.Grow(160 + len(encodedTarget))
	builder.WriteString("CONNECT / HTTP/1.1\r\n")
	// The CNS server selects the first occurrence of Proxy_key, so keep the
	// target-bearing field before camouflage headers (notably when key=Host).
	builder.WriteString(cns.option.Key)
	builder.WriteString(": ")
	builder.WriteString(encodedTarget)
	builder.WriteString("\r\n")
	if udp {
		builder.WriteString(cns.option.Flag)
		builder.WriteString("\r\n")
	}

	headers := make(map[string]string, len(cns.option.Headers)+3)
	headers["Host"] = cns.option.Server
	headers["DNT"] = "1"
	headers["Connection"] = "keep-alive"
	for key, value := range cns.option.Headers {
		if strings.ContainsAny(key, "\r\n") || strings.ContainsAny(value, "\r\n") {
			return "", fmt.Errorf("invalid CNS header %q", key)
		}
		for existingKey := range headers {
			if strings.EqualFold(existingKey, key) {
				delete(headers, existingKey)
			}
		}
		headers[key] = value
	}

	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		builder.WriteString(key)
		builder.WriteString(": ")
		builder.WriteString(headers[key])
		builder.WriteString("\r\n")
	}

	builder.WriteString("\r\n")
	return builder.String(), nil
}

func writeCnsRequest(conn net.Conn, request string) error {
	data := []byte(request)
	for len(data) > 0 {
		n, err := conn.Write(data)
		if n > 0 {
			data = data[n:]
		}
		if err != nil {
			return err
		}
		if n == 0 {
			return errors.New("short CNS handshake write")
		}
	}
	return nil
}

func NewCns(option CnsOption) (*Cns, error) {
	if option.Name == "" {
		return nil, errors.New("missing CNS name")
	}
	if option.Server == "" {
		return nil, errors.New("missing CNS server")
	}
	if option.Port < 1 || option.Port > 65535 {
		return nil, fmt.Errorf("invalid CNS port: %d", option.Port)
	}
	if option.Key == "" {
		option.Key = defaultCnsKey
	}
	if option.Flag == "" {
		option.Flag = defaultCnsFlag
	}
	if strings.ContainsAny(option.Key, "\r\n") {
		return nil, errors.New("invalid CNS key")
	}
	if strings.ContainsAny(option.Flag, "\r\n") {
		return nil, errors.New("invalid CNS UDP flag")
	}

	return &Cns{
		Base: &Base{
			name:   option.Name,
			addr:   net.JoinHostPort(option.Server, strconv.Itoa(option.Port)),
			tp:     C.Cns,
			udp:    option.UDP,
			tfo:    option.TFO,
			mpTcp:  option.MPTCP,
			iface:  option.Interface,
			rmark:  option.RoutingMark,
			prefer: C.NewDNSPrefer(option.IPVersion),
		},
		option: &option,
	}, nil
}
