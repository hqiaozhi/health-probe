package dns

import (
	"context"
	"net"
	"time"
)

type DNS interface {
	Probe(ctx context.Context, in *Request) (*Response, error)
}

type Probe struct{}

type Request struct {
	Target  string
	Server  string
	Timeout int64
}

type Response struct {
	Success  bool
	Ips      []string
	Duration int64
}

func New() DNS {
	return &Probe{}
}

func (l *Probe) Probe(ctx context.Context, in *Request) (*Response, error) {
	start := time.Now()
	timeout := time.Duration(in.Timeout) * time.Second
	if in.Timeout <= 0 {
		timeout = 5 * time.Second // 默认超时5秒
	}

	// 设置超时上下文
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 解析DNS,支持自定义DNS服务器
	if in.Server != "" {
		net.DefaultResolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				return net.DialTimeout(network, in.Server, timeout)
			},
		}
	}
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", in.Target)
	duration := time.Since(start).Milliseconds()
	if err != nil {
		return &Response{
			Success:  false,
			Duration: duration,
		}, err
	}

	// 转换IP为字符串
	ipStrs := make([]string, 0, len(ips))
	for _, ip := range ips {
		ipStrs = append(ipStrs, ip.String())
	}
	return &Response{
		Success:  true,
		Ips:      ipStrs,
		Duration: time.Since(start).Milliseconds(),
	}, nil
}
