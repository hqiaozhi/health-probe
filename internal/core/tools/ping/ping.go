package ping

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"time"
)

type PING interface {
	Probe(ctx context.Context, in *Request) (*Response, error)
}

type Probe struct{}

type Request struct {
	Target  string // 目标IP或域名（必填）
	Timeout int64  // 超时时间（秒），默认5秒
}

type Response struct {
	Success  bool  // 是否探测成功（收到响应）
	Duration int64 // 往返耗时（毫秒）
}

func New() PING {
	return &Probe{}
}

// Probe 执行单次ICMP探测
func (p *Probe) Probe(ctx context.Context, in *Request) (*Response, error) {
	resp := &Response{}

	// 参数默认值设置
	if in.Timeout <= 0 {
		in.Timeout = 5
	}
	if in.Target == "" {
		errMsg := "target is required"
		return resp, errors.New(errMsg)
	}

	// 解析目标IP
	targetIP, err := net.ResolveIPAddr("ip4", in.Target)
	if err != nil {
		errMsg := fmt.Sprintf("resolve target failed: %v", err)
		return resp, errors.New(errMsg)
	}

	// 创建ICMP连接（需要root权限）
	conn, err := net.Dial("ip4:icmp", targetIP.IP.String())
	if err != nil {
		errMsg := fmt.Sprintf("create ICMP connection failed (need root?): %v", err)
		return resp, errors.New(errMsg)
	}
	defer conn.Close()

	// 设置超时（上下文+超时时间取最小）
	timeout := time.Duration(in.Timeout) * time.Second
	deadline := time.Now().Add(timeout)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	if err := conn.SetDeadline(deadline); err != nil {
		errMsg := fmt.Sprintf("set deadline failed: %v", err)
		return resp, errors.New(errMsg)
	}

	// 构造ICMP Echo Request包（单次探测，固定seq=1）
	identifier := os.Getpid() & 0xffff // 用进程ID作为标识
	payload := []byte("icmp probe")    // 简单负载
	packet, err := p.buildICMPPacket(identifier, 1, payload)
	if err != nil {
		errMsg := fmt.Sprintf("build ICMP packet failed: %v", err)
		return resp, errors.New(errMsg)
	}

	// 发送并记录开始时间
	start := time.Now()
	if _, err := conn.Write(packet); err != nil {
		errMsg := fmt.Sprintf("send ICMP packet failed: %v", err)
		resp.Duration = time.Since(start).Milliseconds()
		return resp, errors.New(errMsg)
	}

	// 接收响应
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	resp.Duration = time.Since(start).Milliseconds()

	if err != nil {
		errMsg := fmt.Sprintf("read ICMP response failed: %v", err)
		return resp, errors.New(errMsg)
	}
	// 验证响应有效性
	if err := p.validateICMPResponse(buf[:n], identifier, 1); err != nil {
		errMsg := fmt.Sprintf("invalid ICMP response: %v", err)
		return resp, errors.New(errMsg)
	}

	// 探测成功
	resp.Success = true
	return resp, nil
}

// buildICMPPacket 构造ICMP包（固定Echo Request类型）
func (p *Probe) buildICMPPacket(identifier, seq int, payload []byte) ([]byte, error) {
	packetLen := 8 + len(payload)
	packet := make([]byte, packetLen)

	// ICMP头：类型(8)、代码(0)、校验和(临时0)、标识、序列号
	packet[0] = 8                       // Echo Request类型
	packet[1] = 0                       // 代码
	packet[4] = byte(identifier >> 8)   // 标识高位
	packet[5] = byte(identifier & 0xff) // 标识低位
	packet[6] = byte(seq >> 8)          // 序列号高位
	packet[7] = byte(seq & 0xff)        // 序列号低位
	copy(packet[8:], payload)           // 负载

	// 计算校验和
	checksumHigh, checksumLow := p.calculateChecksum(packet)
	packet[2] = checksumHigh
	packet[3] = checksumLow

	return packet, nil
}

// calculateChecksum 计算ICMP包校验和
func (p *Probe) calculateChecksum(data []byte) (uint8, uint8) {
	var sum uint32
	for i := 0; i < len(data); i += 2 {
		if i+1 < len(data) {
			sum += uint32(binary.BigEndian.Uint16(data[i:]))
		} else {
			sum += uint32(data[i]) << 8
		}
	}

	// 折叠校验和并取反
	for sum>>16 != 0 {
		sum = (sum & 0xffff) + (sum >> 16)
	}
	checksum := ^uint16(sum)
	return uint8(checksum >> 8), uint8(checksum & 0xff)
}

// validateICMPResponse 验证ICMP响应（只检查关键字段）
func (p *Probe) validateICMPResponse(data []byte, identifier, seq int) error {
	// 最小长度：IP头(20) + ICMP头(8)
	if len(data) < 28 {
		return fmt.Errorf("response too short")
	}

	// 跳过IP头，获取ICMP响应部分
	ipHeaderLen := int(data[0]&0x0f) * 4
	icmpResp := data[ipHeaderLen:]

	// 验证类型（Echo Reply应该是0）、标识、序列号
	if icmpResp[0] != 0 {
		return fmt.Errorf("invalid type: %d (expected 0)", icmpResp[0])
	}
	respID := int(icmpResp[4])<<8 | int(icmpResp[5])
	if respID != identifier {
		return fmt.Errorf("identifier mismatch")
	}
	respSeq := int(icmpResp[6])<<8 | int(icmpResp[7])
	if respSeq != seq {
		return fmt.Errorf("sequence mismatch")
	}

	return nil
}
