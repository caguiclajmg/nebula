package nebula

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

//var ips []uint32 = []uint32{9000, 9999999, 3, 292394923}
var ips []uint32

func Test_NewHandshakeManagerIndex(t *testing.T) {
	l := NewTestLogger()
	_, tuncidr, _ := net.ParseCIDR("172.1.1.1/24")
	_, vpncidr, _ := net.ParseCIDR("172.1.1.1/24")
	_, localrange, _ := net.ParseCIDR("10.1.1.1/24")
	ips = []uint32{ip2int(net.ParseIP("172.1.1.2"))}
	preferredRanges := []*net.IPNet{localrange}
	mainHM := NewHostMap(l, "test", vpncidr, preferredRanges)

	blah := NewHandshakeManager(l, tuncidr, preferredRanges, mainHM, &LightHouse{}, &udpConn{}, defaultHandshakeConfig)

	now := time.Now()
	blah.NextInboundHandshakeTimerTick(now)

	var indexes = make([]uint32, 4)
	var hostinfo = make([]*HostInfo, len(indexes))
	for i := range indexes {
		hostinfo[i] = &HostInfo{ConnectionState: &ConnectionState{}}
	}

	// Add four indexes
	for i := range indexes {
		err := blah.AddIndexHostInfo(hostinfo[i])
		assert.NoError(t, err)
		indexes[i] = hostinfo[i].localIndexId
		blah.InboundHandshakeTimer.Add(indexes[i], time.Second*10)
	}
	// Confirm they are in the pending index list
	for _, v := range indexes {
		assert.Contains(t, blah.pendingHostMap.Indexes, uint32(v))
	}
	// Adding something to pending should not affect the main hostmap
	assert.Len(t, mainHM.Indexes, 0)
	// Jump ahead 8 seconds
	for i := 1; i <= DefaultHandshakeRetries; i++ {
		next_tick := now.Add(DefaultHandshakeTryInterval * time.Duration(i))
		blah.NextInboundHandshakeTimerTick(next_tick)
	}
	// Confirm they are still in the pending index list
	for _, v := range indexes {
		assert.Contains(t, blah.pendingHostMap.Indexes, uint32(v))
	}
	// Jump ahead 4 more seconds
	next_tick := now.Add(12 * time.Second)
	blah.NextInboundHandshakeTimerTick(next_tick)
	// Confirm they have been removed
	for _, v := range indexes {
		assert.NotContains(t, blah.pendingHostMap.Indexes, uint32(v))
	}
}

func Test_NewHandshakeManagerVpnIP(t *testing.T) {
	l := NewTestLogger()
	_, tuncidr, _ := net.ParseCIDR("172.1.1.1/24")
	_, vpncidr, _ := net.ParseCIDR("172.1.1.1/24")
	_, localrange, _ := net.ParseCIDR("10.1.1.1/24")
	ips = []uint32{ip2int(net.ParseIP("172.1.1.2"))}
	preferredRanges := []*net.IPNet{localrange}
	mw := &mockEncWriter{}
	mainHM := NewHostMap(l, "test", vpncidr, preferredRanges)

	blah := NewHandshakeManager(l, tuncidr, preferredRanges, mainHM, &LightHouse{}, &udpConn{}, defaultHandshakeConfig)

	now := time.Now()
	blah.NextOutboundHandshakeTimerTick(now, mw)

	// Add four "IPs" - which are just uint32s
	for _, v := range ips {
		blah.AddVpnIP(v)
	}
	// Adding something to pending should not affect the main hostmap
	assert.Len(t, mainHM.Hosts, 0)
	// Confirm they are in the pending index list
	for _, v := range ips {
		assert.Contains(t, blah.pendingHostMap.Hosts, uint32(v))
	}

	// Jump ahead `HandshakeRetries` ticks
	cumulative := time.Duration(0)
	for i := 0; i <= DefaultHandshakeRetries+1; i++ {
		cumulative += time.Duration(i)*DefaultHandshakeTryInterval + 1
		next_tick := now.Add(cumulative)
		//l.Infoln(next_tick)
		blah.NextOutboundHandshakeTimerTick(next_tick, mw)
	}

	// Confirm they are still in the pending index list
	for _, v := range ips {
		assert.Contains(t, blah.pendingHostMap.Hosts, uint32(v))
	}
	// Jump ahead 1 more second
	cumulative += time.Duration(DefaultHandshakeRetries+1) * DefaultHandshakeTryInterval
	next_tick := now.Add(cumulative)
	//l.Infoln(next_tick)
	blah.NextOutboundHandshakeTimerTick(next_tick, mw)
	// Confirm they have been removed
	for _, v := range ips {
		assert.NotContains(t, blah.pendingHostMap.Hosts, uint32(v))
	}
}

func Test_NewHandshakeManagerTrigger(t *testing.T) {
	l := NewTestLogger()
	_, tuncidr, _ := net.ParseCIDR("172.1.1.1/24")
	_, vpncidr, _ := net.ParseCIDR("172.1.1.1/24")
	_, localrange, _ := net.ParseCIDR("10.1.1.1/24")
	ip := ip2int(net.ParseIP("172.1.1.2"))
	preferredRanges := []*net.IPNet{localrange}
	mw := &mockEncWriter{}
	mainHM := NewHostMap(l, "test", vpncidr, preferredRanges)
	lh := &LightHouse{}

	blah := NewHandshakeManager(l, tuncidr, preferredRanges, mainHM, lh, &udpConn{}, defaultHandshakeConfig)

	now := time.Now()
	blah.NextOutboundHandshakeTimerTick(now, mw)

	assert.Equal(t, 0, testCountTimerWheelEntries(blah.OutboundHandshakeTimer))

	blah.AddVpnIP(ip)

	assert.Equal(t, 1, testCountTimerWheelEntries(blah.OutboundHandshakeTimer))

	// Trigger the same method the channel will
	blah.handleOutbound(ip, mw, true)

	// Make sure the trigger doesn't schedule another timer entry
	assert.Equal(t, 1, testCountTimerWheelEntries(blah.OutboundHandshakeTimer))
	hi := blah.pendingHostMap.Hosts[ip]
	assert.Nil(t, hi.remote)

	uaddr := NewUDPAddrFromString("10.1.1.1:4242")
	lh.addrMap = map[uint32]*ip4And6{}
	lh.addrMap[ip] = &ip4And6{
		v4: []*Ip4AndPort{NewIp4AndPort(uaddr.IP, uint32(uaddr.Port))},
		v6: []*Ip6AndPort{},
	}

	// This should trigger the hostmap to populate the hostinfo
	blah.handleOutbound(ip, mw, true)
	assert.NotNil(t, hi.remote)
	assert.Equal(t, 1, testCountTimerWheelEntries(blah.OutboundHandshakeTimer))
}

func testCountTimerWheelEntries(tw *SystemTimerWheel) (c int) {
	for _, i := range tw.wheel {
		n := i.Head
		for n != nil {
			c++
			n = n.Next
		}
	}
	return c
}

func Test_NewHandshakeManagerVpnIPcleanup(t *testing.T) {
	l := NewTestLogger()
	_, tuncidr, _ := net.ParseCIDR("172.1.1.1/24")
	_, vpncidr, _ := net.ParseCIDR("172.1.1.1/24")
	_, localrange, _ := net.ParseCIDR("10.1.1.1/24")
	vpnIP = ip2int(net.ParseIP("172.1.1.2"))
	preferredRanges := []*net.IPNet{localrange}
	mw := &mockEncWriter{}
	mainHM := NewHostMap(l, "test", vpncidr, preferredRanges)

	blah := NewHandshakeManager(l, tuncidr, preferredRanges, mainHM, &LightHouse{}, &udpConn{}, defaultHandshakeConfig)

	now := time.Now()
	blah.NextOutboundHandshakeTimerTick(now, mw)

	hostinfo := blah.AddVpnIP(vpnIP)
	// Pretned we have an index too
	err := blah.AddIndexHostInfo(hostinfo)
	assert.NoError(t, err)
	blah.InboundHandshakeTimer.Add(hostinfo.localIndexId, time.Second*10)
	assert.NotZero(t, hostinfo.localIndexId)
	assert.Contains(t, blah.pendingHostMap.Indexes, hostinfo.localIndexId)

	// Jump ahead `HandshakeRetries` ticks. Eviction should happen in pending
	// but not main hostmap
	cumulative := time.Duration(0)
	for i := 1; i <= DefaultHandshakeRetries+2; i++ {
		cumulative += DefaultHandshakeTryInterval * time.Duration(i)
		next_tick := now.Add(cumulative)
		blah.NextOutboundHandshakeTimerTick(next_tick, mw)
	}
	/*
		for i := 0; i <= HandshakeRetries+1; i++ {
			next_tick := now.Add(cumulative)
			//l.Infoln(next_tick)
			blah.NextOutboundHandshakeTimerTick(next_tick)
		}
	*/
	/*
		for i := 0; i <= HandshakeRetries+1; i++ {
			next_tick := now.Add(time.Duration(i) * time.Second)
			blah.NextOutboundHandshakeTimerTick(next_tick)
		}
	*/

	/*
		cumulative += HandshakeTryInterval*time.Duration(HandshakeRetries) + 3
		next_tick := now.Add(cumulative)
		l.Infoln(cumulative, next_tick)
		blah.NextOutboundHandshakeTimerTick(next_tick)
	*/
	assert.NotContains(t, blah.pendingHostMap.Hosts, uint32(vpnIP))
	assert.NotContains(t, blah.pendingHostMap.Indexes, uint32(12341234))
}

func Test_NewHandshakeManagerIndexcleanup(t *testing.T) {
	l := NewTestLogger()
	_, tuncidr, _ := net.ParseCIDR("172.1.1.1/24")
	_, vpncidr, _ := net.ParseCIDR("172.1.1.1/24")
	_, localrange, _ := net.ParseCIDR("10.1.1.1/24")
	preferredRanges := []*net.IPNet{localrange}
	mainHM := NewHostMap(l, "test", vpncidr, preferredRanges)

	blah := NewHandshakeManager(l, tuncidr, preferredRanges, mainHM, &LightHouse{}, &udpConn{}, defaultHandshakeConfig)

	now := time.Now()
	blah.NextInboundHandshakeTimerTick(now)

	hostinfo := &HostInfo{ConnectionState: &ConnectionState{}}
	err := blah.AddIndexHostInfo(hostinfo)
	assert.NoError(t, err)
	blah.InboundHandshakeTimer.Add(hostinfo.localIndexId, time.Second*10)
	// Pretned we have an index too
	blah.pendingHostMap.AddVpnIPHostInfo(101010, hostinfo)
	assert.Contains(t, blah.pendingHostMap.Hosts, uint32(101010))

	for i := 1; i <= DefaultHandshakeRetries+2; i++ {
		next_tick := now.Add(DefaultHandshakeTryInterval * time.Duration(i))
		blah.NextInboundHandshakeTimerTick(next_tick)
	}

	next_tick := now.Add(DefaultHandshakeTryInterval*DefaultHandshakeRetries + 3)
	blah.NextInboundHandshakeTimerTick(next_tick)
	assert.NotContains(t, blah.pendingHostMap.Hosts, uint32(101010))
	assert.NotContains(t, blah.pendingHostMap.Indexes, uint32(hostinfo.localIndexId))
}

type mockEncWriter struct {
}

func (mw *mockEncWriter) SendMessageToVpnIp(t NebulaMessageType, st NebulaMessageSubType, vpnIp uint32, p, nb, out []byte) {
	return
}

func (mw *mockEncWriter) SendMessageToAll(t NebulaMessageType, st NebulaMessageSubType, vpnIp uint32, p, nb, out []byte) {
	return
}
