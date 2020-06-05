package outbound

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/Dreamacro/clash/common/structure"
	"github.com/Dreamacro/clash/component/dialer"
	obfs "github.com/Dreamacro/clash/component/simple-obfs"
	"github.com/Dreamacro/clash/component/snell"
	C "github.com/Dreamacro/clash/constant"
)

type Snell struct {
	*Base
	psk        []byte
	obfsOption *simpleObfsOption
}

type SnellOption struct {
	Name     string                 `proxy:"name"`
	Server   string                 `proxy:"server"`
	Port     int                    `proxy:"port"`
	Psk      string                 `proxy:"psk"`
	ObfsOpts map[string]interface{} `proxy:"obfs-opts,omitempty"`
}

func (s *Snell) StreamConn(c net.Conn, metadata *C.Metadata) (net.Conn, error) {
	switch s.obfsOption.Mode {
	case "tls":
		c = obfs.NewTLSObfs(c, s.obfsOption.Host)
	case "http":
		_, port, _ := net.SplitHostPort(s.addr)
		c = obfs.NewHTTPObfs(c, s.obfsOption.Host, port)
	}
	c = snell.StreamConn(c, s.psk)
	port, _ := strconv.Atoi(metadata.DstPort)
	err := snell.WriteHeader(c, metadata.String(), uint(port))
	return c, err
}

func (s *Snell) DialContext(ctx context.Context, metadata *C.Metadata) (C.Conn, error) {
	c, err := dialer.DialContext(ctx, "tcp", s.addr)
	if err != nil {
		return nil, fmt.Errorf("%s connect error: %w", s.addr, err)
	}
	tcpKeepAlive(c)

	c, err = s.StreamConn(c, metadata)
	return NewConn(c, s), err
}

func NewSnell(option SnellOption) (*Snell, error) {
	addr := net.JoinHostPort(option.Server, strconv.Itoa(option.Port))
	psk := []byte(option.Psk)

	decoder := structure.NewDecoder(structure.Option{TagName: "obfs", WeaklyTypedInput: true})
	obfsOption := &simpleObfsOption{Host: "bing.com"}
	if err := decoder.Decode(option.ObfsOpts, obfsOption); err != nil {
		return nil, fmt.Errorf("snell %s initialize obfs error: %w", addr, err)
	}

	if obfsOption.Mode != "tls" && obfsOption.Mode != "http" {
		return nil, fmt.Errorf("snell %s obfs mode error: %s", addr, obfsOption.Mode)
	}

	return &Snell{
		Base: &Base{
			name: option.Name,
			addr: addr,
			tp:   C.Snell,
		},
		psk:        psk,
		obfsOption: obfsOption,
	}, nil
}
