package nebula

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rcrowley/go-metrics"
	"github.com/sirupsen/logrus"
	"github.com/slackhq/nebula/cert"
)

//const ProbeLen = 100
const PromoteEvery = 1000
const MaxRemotes = 10

// How long we should prevent roaming back to the previous IP.
// This helps prevent flapping due to packets already in flight
const RoamingSuppressSeconds = 2

type HostMap struct {
	sync.RWMutex    //Because we concurrently read and write to our maps
	name            string
	Indexes         map[uint32]*HostInfo
	RemoteIndexes   map[uint32]*HostInfo
	Hosts           map[uint32]*HostInfo
	preferredRanges []*net.IPNet
	vpnCIDR         *net.IPNet
	defaultRoute    uint32
	unsafeRoutes    *CIDRTree
	metricsEnabled  bool
	l               *logrus.Logger
}

type HostInfo struct {
	sync.RWMutex

	remote            *udpAddr
	Remotes           []*udpAddr
	promoteCounter    uint32
	ConnectionState   *ConnectionState
	handshakeStart    time.Time
	HandshakeReady    bool
	HandshakeCounter  int
	HandshakeComplete bool
	HandshakePacket   map[uint8][]byte
	packetStore       []*cachedPacket
	remoteIndexId     uint32
	localIndexId      uint32
	hostId            uint32
	recvError         int
	remoteCidr        *CIDRTree

	// This is a list of remotes that we have tried to handshake with and have returned from the wrong vpn ip.
	// They should not be tried again during a handshake
	badRemotes []*udpAddr

	// lastRebindCount is the other side of Interface.rebindCount, if these values don't match then we need to ask LH
	// for a punch from the remote end of this tunnel. The goal being to prime their conntrack for our traffic just like
	// with a handshake
	lastRebindCount int8

	lastRoam       time.Time
	lastRoamRemote *udpAddr
}

type cachedPacket struct {
	messageType    NebulaMessageType
	messageSubType NebulaMessageSubType
	callback       packetCallback
	packet         []byte
}

type packetCallback func(t NebulaMessageType, st NebulaMessageSubType, h *HostInfo, p, nb, out []byte)

func NewHostMap(l *logrus.Logger, name string, vpnCIDR *net.IPNet, preferredRanges []*net.IPNet) *HostMap {
	h := map[uint32]*HostInfo{}
	i := map[uint32]*HostInfo{}
	r := map[uint32]*HostInfo{}
	m := HostMap{
		name:            name,
		Indexes:         i,
		RemoteIndexes:   r,
		Hosts:           h,
		preferredRanges: preferredRanges,
		vpnCIDR:         vpnCIDR,
		defaultRoute:    0,
		unsafeRoutes:    NewCIDRTree(),
		l:               l,
	}
	return &m
}

// UpdateStats takes a name and reports host and index counts to the stats collection system
func (hm *HostMap) EmitStats(name string) {
	hm.RLock()
	hostLen := len(hm.Hosts)
	indexLen := len(hm.Indexes)
	remoteIndexLen := len(hm.RemoteIndexes)
	hm.RUnlock()

	metrics.GetOrRegisterGauge("hostmap."+name+".hosts", nil).Update(int64(hostLen))
	metrics.GetOrRegisterGauge("hostmap."+name+".indexes", nil).Update(int64(indexLen))
	metrics.GetOrRegisterGauge("hostmap."+name+".remoteIndexes", nil).Update(int64(remoteIndexLen))
}

func (hm *HostMap) GetIndexByVpnIP(vpnIP uint32) (uint32, error) {
	hm.RLock()
	if i, ok := hm.Hosts[vpnIP]; ok {
		index := i.localIndexId
		hm.RUnlock()
		return index, nil
	}
	hm.RUnlock()
	return 0, errors.New("vpn IP not found")
}

func (hm *HostMap) Add(ip uint32, hostinfo *HostInfo) {
	hm.Lock()
	hm.Hosts[ip] = hostinfo
	hm.Unlock()
}

func (hm *HostMap) AddVpnIP(vpnIP uint32) *HostInfo {
	h := &HostInfo{}
	hm.RLock()
	if _, ok := hm.Hosts[vpnIP]; !ok {
		hm.RUnlock()
		h = &HostInfo{
			Remotes:         []*udpAddr{},
			promoteCounter:  0,
			hostId:          vpnIP,
			HandshakePacket: make(map[uint8][]byte, 0),
		}
		hm.Lock()
		hm.Hosts[vpnIP] = h
		hm.Unlock()
		return h
	} else {
		h = hm.Hosts[vpnIP]
		hm.RUnlock()
		return h
	}
}

func (hm *HostMap) DeleteVpnIP(vpnIP uint32) {
	hm.Lock()
	delete(hm.Hosts, vpnIP)
	if len(hm.Hosts) == 0 {
		hm.Hosts = map[uint32]*HostInfo{}
	}
	hm.Unlock()

	if hm.l.Level >= logrus.DebugLevel {
		hm.l.WithField("hostMap", m{"mapName": hm.name, "vpnIp": IntIp(vpnIP), "mapTotalSize": len(hm.Hosts)}).
			Debug("Hostmap vpnIp deleted")
	}
}

// Only used by pendingHostMap when the remote index is not initially known
func (hm *HostMap) addRemoteIndexHostInfo(index uint32, h *HostInfo) {
	hm.Lock()
	h.remoteIndexId = index
	hm.RemoteIndexes[index] = h
	hm.Unlock()

	if hm.l.Level > logrus.DebugLevel {
		hm.l.WithField("hostMap", m{"mapName": hm.name, "indexNumber": index, "mapTotalSize": len(hm.Indexes),
			"hostinfo": m{"existing": true, "localIndexId": h.localIndexId, "hostId": IntIp(h.hostId)}}).
			Debug("Hostmap remoteIndex added")
	}
}

func (hm *HostMap) AddVpnIPHostInfo(vpnIP uint32, h *HostInfo) {
	hm.Lock()
	h.hostId = vpnIP
	hm.Hosts[vpnIP] = h
	hm.Indexes[h.localIndexId] = h
	hm.RemoteIndexes[h.remoteIndexId] = h
	hm.Unlock()

	if hm.l.Level > logrus.DebugLevel {
		hm.l.WithField("hostMap", m{"mapName": hm.name, "vpnIp": IntIp(vpnIP), "mapTotalSize": len(hm.Hosts),
			"hostinfo": m{"existing": true, "localIndexId": h.localIndexId, "hostId": IntIp(h.hostId)}}).
			Debug("Hostmap vpnIp added")
	}
}

// This is only called in pendingHostmap, to cleanup an inbound handshake
func (hm *HostMap) DeleteIndex(index uint32) {
	hm.Lock()
	hostinfo, ok := hm.Indexes[index]
	if ok {
		delete(hm.Indexes, index)
		delete(hm.RemoteIndexes, hostinfo.remoteIndexId)

		// Check if we have an entry under hostId that matches the same hostinfo
		// instance. Clean it up as well if we do.
		hostinfo2, ok := hm.Hosts[hostinfo.hostId]
		if ok && hostinfo2 == hostinfo {
			delete(hm.Hosts, hostinfo.hostId)
		}
	}
	hm.Unlock()

	if hm.l.Level >= logrus.DebugLevel {
		hm.l.WithField("hostMap", m{"mapName": hm.name, "indexNumber": index, "mapTotalSize": len(hm.Indexes)}).
			Debug("Hostmap index deleted")
	}
}

// This is used to cleanup on recv_error
func (hm *HostMap) DeleteReverseIndex(index uint32) {
	hm.Lock()
	hostinfo, ok := hm.RemoteIndexes[index]
	if ok {
		delete(hm.Indexes, hostinfo.localIndexId)
		delete(hm.RemoteIndexes, index)

		// Check if we have an entry under hostId that matches the same hostinfo
		// instance. Clean it up as well if we do (they might not match in pendingHostmap)
		var hostinfo2 *HostInfo
		hostinfo2, ok = hm.Hosts[hostinfo.hostId]
		if ok && hostinfo2 == hostinfo {
			delete(hm.Hosts, hostinfo.hostId)
		}
	}
	hm.Unlock()

	if hm.l.Level >= logrus.DebugLevel {
		hm.l.WithField("hostMap", m{"mapName": hm.name, "indexNumber": index, "mapTotalSize": len(hm.Indexes)}).
			Debug("Hostmap remote index deleted")
	}
}

func (hm *HostMap) DeleteHostInfo(hostinfo *HostInfo) {
	hm.Lock()

	// Check if this same hostId is in the hostmap with a different instance.
	// This could happen if we have an entry in the pending hostmap with different
	// index values than the one in the main hostmap.
	hostinfo2, ok := hm.Hosts[hostinfo.hostId]
	if ok && hostinfo2 != hostinfo {
		delete(hm.Hosts, hostinfo2.hostId)
		delete(hm.Indexes, hostinfo2.localIndexId)
		delete(hm.RemoteIndexes, hostinfo2.remoteIndexId)
	}

	delete(hm.Hosts, hostinfo.hostId)
	if len(hm.Hosts) == 0 {
		hm.Hosts = map[uint32]*HostInfo{}
	}
	delete(hm.Indexes, hostinfo.localIndexId)
	if len(hm.Indexes) == 0 {
		hm.Indexes = map[uint32]*HostInfo{}
	}
	delete(hm.RemoteIndexes, hostinfo.remoteIndexId)
	if len(hm.RemoteIndexes) == 0 {
		hm.RemoteIndexes = map[uint32]*HostInfo{}
	}
	hm.Unlock()

	if hm.l.Level >= logrus.DebugLevel {
		hm.l.WithField("hostMap", m{"mapName": hm.name, "mapTotalSize": len(hm.Hosts),
			"vpnIp": IntIp(hostinfo.hostId), "indexNumber": hostinfo.localIndexId, "remoteIndexNumber": hostinfo.remoteIndexId}).
			Debug("Hostmap hostInfo deleted")
	}
}

func (hm *HostMap) QueryIndex(index uint32) (*HostInfo, error) {
	//TODO: we probably just want ot return bool instead of error, or at least a static error
	hm.RLock()
	if h, ok := hm.Indexes[index]; ok {
		hm.RUnlock()
		return h, nil
	} else {
		hm.RUnlock()
		return nil, errors.New("unable to find index")
	}
}

func (hm *HostMap) QueryReverseIndex(index uint32) (*HostInfo, error) {
	hm.RLock()
	if h, ok := hm.RemoteIndexes[index]; ok {
		hm.RUnlock()
		return h, nil
	} else {
		hm.RUnlock()
		return nil, fmt.Errorf("unable to find reverse index or connectionstate nil in %s hostmap", hm.name)
	}
}

func (hm *HostMap) AddRemote(vpnIp uint32, remote *udpAddr) *HostInfo {
	hm.Lock()
	i, v := hm.Hosts[vpnIp]
	if v {
		i.AddRemote(remote)
	} else {
		i = &HostInfo{
			Remotes:         []*udpAddr{remote.Copy()},
			promoteCounter:  0,
			hostId:          vpnIp,
			HandshakePacket: make(map[uint8][]byte, 0),
		}
		i.remote = i.Remotes[0]
		hm.Hosts[vpnIp] = i
		if hm.l.Level >= logrus.DebugLevel {
			hm.l.WithField("hostMap", m{"mapName": hm.name, "vpnIp": IntIp(vpnIp), "udpAddr": remote, "mapTotalSize": len(hm.Hosts)}).
				Debug("Hostmap remote ip added")
		}
	}
	i.ForcePromoteBest(hm.preferredRanges)
	hm.Unlock()
	return i
}

func (hm *HostMap) QueryVpnIP(vpnIp uint32) (*HostInfo, error) {
	return hm.queryVpnIP(vpnIp, nil)
}

// PromoteBestQueryVpnIP will attempt to lazily switch to the best remote every
// `PromoteEvery` calls to this function for a given host.
func (hm *HostMap) PromoteBestQueryVpnIP(vpnIp uint32, ifce *Interface) (*HostInfo, error) {
	return hm.queryVpnIP(vpnIp, ifce)
}

func (hm *HostMap) queryVpnIP(vpnIp uint32, promoteIfce *Interface) (*HostInfo, error) {
	hm.RLock()
	if h, ok := hm.Hosts[vpnIp]; ok {
		if promoteIfce != nil {
			h.TryPromoteBest(hm.preferredRanges, promoteIfce)
		}
		//fmt.Println(h.remote)
		hm.RUnlock()
		return h, nil
	} else {
		//return &net.UDPAddr{}, nil, errors.New("Unable to find host")
		hm.RUnlock()
		/*
			if lightHouse != nil {
				lightHouse.Query(vpnIp)
				return nil, errors.New("Unable to find host")
			}
		*/
		return nil, errors.New("unable to find host")
	}
}

func (hm *HostMap) queryUnsafeRoute(ip uint32) uint32 {
	r := hm.unsafeRoutes.MostSpecificContains(ip)
	if r != nil {
		return r.(uint32)
	} else {
		return 0
	}
}

// We already have the hm Lock when this is called, so make sure to not call
// any other methods that might try to grab it again
func (hm *HostMap) addHostInfo(hostinfo *HostInfo, f *Interface) {
	remoteCert := hostinfo.ConnectionState.peerCert
	ip := ip2int(remoteCert.Details.Ips[0].IP)

	f.lightHouse.AddRemoteAndReset(ip, hostinfo.remote)
	if f.serveDns {
		for i := range remoteCert.Details.Names {
			dnsR.Add(remoteCert.Details.Names[i]+".", remoteCert.Details.Ips[0].IP.String())
		}
	}

	hm.Hosts[hostinfo.hostId] = hostinfo
	hm.Indexes[hostinfo.localIndexId] = hostinfo
	hm.RemoteIndexes[hostinfo.remoteIndexId] = hostinfo

	if hm.l.Level >= logrus.DebugLevel {
		hm.l.WithField("hostMap", m{"mapName": hm.name, "vpnIp": IntIp(hostinfo.hostId), "mapTotalSize": len(hm.Hosts),
			"hostinfo": m{"existing": true, "localIndexId": hostinfo.localIndexId, "hostId": IntIp(hostinfo.hostId)}}).
			Debug("Hostmap vpnIp added")
	}
}

func (hm *HostMap) ClearRemotes(vpnIP uint32) {
	hm.Lock()
	i := hm.Hosts[vpnIP]
	if i == nil {
		hm.Unlock()
		return
	}
	i.remote = nil
	i.Remotes = nil
	hm.Unlock()
}

func (hm *HostMap) SetDefaultRoute(ip uint32) {
	hm.defaultRoute = ip
}

func (hm *HostMap) PunchList() []*udpAddr {
	var list []*udpAddr
	hm.RLock()
	for _, v := range hm.Hosts {
		for _, r := range v.Remotes {
			list = append(list, r)
		}
		//	if h, ok := hm.Hosts[vpnIp]; ok {
		//		hm.Hosts[vpnIp].PromoteBest(hm.preferredRanges, false)
		//fmt.Println(h.remote)
		//	}
	}
	hm.RUnlock()
	return list
}

func (hm *HostMap) Punchy(conn *udpConn) {
	var metricsTxPunchy metrics.Counter
	if hm.metricsEnabled {
		metricsTxPunchy = metrics.GetOrRegisterCounter("messages.tx.punchy", nil)
	} else {
		metricsTxPunchy = metrics.NilCounter{}
	}

	b := []byte{1}
	for {
		for _, addr := range hm.PunchList() {
			metricsTxPunchy.Inc(1)
			conn.WriteTo(b, addr)
		}
		time.Sleep(time.Second * 30)
	}
}

func (hm *HostMap) addUnsafeRoutes(routes *[]route) {
	for _, r := range *routes {
		hm.l.WithField("route", r.route).WithField("via", r.via).Warn("Adding UNSAFE Route")
		hm.unsafeRoutes.AddCIDR(r.route, ip2int(*r.via))
	}
}

func (i *HostInfo) MarshalJSON() ([]byte, error) {
	return json.Marshal(m{
		"remote":             i.remote,
		"remotes":            i.Remotes,
		"promote_counter":    i.promoteCounter,
		"connection_state":   i.ConnectionState,
		"handshake_start":    i.handshakeStart,
		"handshake_ready":    i.HandshakeReady,
		"handshake_counter":  i.HandshakeCounter,
		"handshake_complete": i.HandshakeComplete,
		"handshake_packet":   i.HandshakePacket,
		"packet_store":       i.packetStore,
		"remote_index":       i.remoteIndexId,
		"local_index":        i.localIndexId,
		"host_id":            int2ip(i.hostId),
		"receive_errors":     i.recvError,
		"last_roam":          i.lastRoam,
		"last_roam_remote":   i.lastRoamRemote,
	})
}

func (i *HostInfo) BindConnectionState(cs *ConnectionState) {
	i.ConnectionState = cs
}

func (i *HostInfo) TryPromoteBest(preferredRanges []*net.IPNet, ifce *Interface) {
	if i.remote == nil {
		i.ForcePromoteBest(preferredRanges)
		return
	}

	if atomic.AddUint32(&i.promoteCounter, 1)%PromoteEvery == 0 {
		// return early if we are already on a preferred remote
		rIP := i.remote.IP
		for _, l := range preferredRanges {
			if l.Contains(rIP) {
				return
			}
		}

		// We re-query the lighthouse periodically while sending packets, so
		// check for new remotes in our local lighthouse cache
		ips := ifce.lightHouse.QueryCache(i.hostId)
		for _, ip := range ips {
			i.AddRemote(ip)
		}

		best, preferred := i.getBestRemote(preferredRanges)
		if preferred && !best.Equals(i.remote) {
			// Try to send a test packet to that host, this should
			// cause it to detect a roaming event and switch remotes
			ifce.send(test, testRequest, i.ConnectionState, i, best, []byte(""), make([]byte, 12, 12), make([]byte, mtu))
		}
	}
}

func (i *HostInfo) ForcePromoteBest(preferredRanges []*net.IPNet) {
	best, _ := i.getBestRemote(preferredRanges)
	if best != nil {
		i.remote = best
	}
}

func (i *HostInfo) getBestRemote(preferredRanges []*net.IPNet) (best *udpAddr, preferred bool) {
	if len(i.Remotes) > 0 {
		for _, r := range i.Remotes {
			for _, l := range preferredRanges {
				if l.Contains(r.IP) {
					return r, true
				}
			}

			if best == nil || !PrivateIP(r.IP) {
				best = r
			}
			/*
				for _, r := range i.Remotes {
					// Must have > 80% probe success to be considered.
					//fmt.Println("GRADE:", r.addr.IP, r.Grade())
					if r.Grade() > float64(.8) {
						if localToMe.Contains(r.addr.IP) == true {
							best = r.addr
							break
							//i.remote = i.Remotes[c].addr
						} else {
								//}
					}
			*/
		}
		return best, false
	}

	return nil, false
}

// rotateRemote will move remote to the next ip in the list of remote ips for this host
// This is different than PromoteBest in that what is algorithmically best may not actually work.
// Only known use case is when sending a stage 0 handshake.
// It may be better to just send stage 0 handshakes to all known ips and sort it out in the receiver.
func (i *HostInfo) rotateRemote() {
	// We have 0, can't rotate
	if len(i.Remotes) < 1 {
		return
	}

	if i.remote == nil {
		i.remote = i.Remotes[0]
		return
	}

	// We want to look at all but the very last entry since that is handled at the end
	for x := 0; x < len(i.Remotes)-1; x++ {
		// Find our current position and move to the next one in the list
		if i.Remotes[x].Equals(i.remote) {
			i.remote = i.Remotes[x+1]
			return
		}
	}

	// Our current position was likely the last in the list, start over at 0
	i.remote = i.Remotes[0]
}

func (i *HostInfo) cachePacket(l *logrus.Logger, t NebulaMessageType, st NebulaMessageSubType, packet []byte, f packetCallback) {
	//TODO: return the error so we can log with more context
	if len(i.packetStore) < 100 {
		tempPacket := make([]byte, len(packet))
		copy(tempPacket, packet)
		//l.WithField("trace", string(debug.Stack())).Error("Caching packet", tempPacket)
		i.packetStore = append(i.packetStore, &cachedPacket{t, st, f, tempPacket})
		if l.Level >= logrus.DebugLevel {
			i.logger(l).
				WithField("length", len(i.packetStore)).
				WithField("stored", true).
				Debugf("Packet store")
		}

	} else if l.Level >= logrus.DebugLevel {
		i.logger(l).
			WithField("length", len(i.packetStore)).
			WithField("stored", false).
			Debugf("Packet store")
	}
}

// handshakeComplete will set the connection as ready to communicate, as well as flush any stored packets
func (i *HostInfo) handshakeComplete(l *logrus.Logger) {
	//TODO: I'm not certain the distinction between handshake complete and ConnectionState being ready matters because:
	//TODO: HandshakeComplete means send stored packets and ConnectionState.ready means we are ready to send
	//TODO: if the transition from HandhsakeComplete to ConnectionState.ready happens all within this function they are identical

	i.ConnectionState.queueLock.Lock()
	i.HandshakeComplete = true
	//TODO: this should be managed by the handshake state machine to set it based on how many handshake were seen.
	// Clamping it to 2 gets us out of the woods for now
	atomic.StoreUint64(&i.ConnectionState.atomicMessageCounter, 2)

	if l.Level >= logrus.DebugLevel {
		i.logger(l).Debugf("Sending %d stored packets", len(i.packetStore))
	}

	if len(i.packetStore) > 0 {
		nb := make([]byte, 12, 12)
		out := make([]byte, mtu)
		for _, cp := range i.packetStore {
			cp.callback(cp.messageType, cp.messageSubType, i, cp.packet, nb, out)
		}
	}

	i.badRemotes = make([]*udpAddr, 0)
	i.packetStore = make([]*cachedPacket, 0)
	i.ConnectionState.ready = true
	i.ConnectionState.queueLock.Unlock()
	i.ConnectionState.certState = nil
}

func (i *HostInfo) CopyRemotes() []*udpAddr {
	i.RLock()
	rc := make([]*udpAddr, len(i.Remotes), len(i.Remotes))
	for x, addr := range i.Remotes {
		rc[x] = addr.Copy()
	}
	i.RUnlock()
	return rc
}

func (i *HostInfo) GetCert() *cert.NebulaCertificate {
	if i.ConnectionState != nil {
		return i.ConnectionState.peerCert
	}
	return nil
}

func (i *HostInfo) AddRemote(remote *udpAddr) *udpAddr {
	if i.unlockedIsBadRemote(remote) {
		return i.remote
	}

	for _, r := range i.Remotes {
		if r.Equals(remote) {
			return r
		}
	}

	// Trim this down if necessary
	if len(i.Remotes) > MaxRemotes {
		i.Remotes = i.Remotes[len(i.Remotes)-MaxRemotes:]
	}

	rc := remote.Copy()
	i.Remotes = append(i.Remotes, rc)
	return rc
}

func (i *HostInfo) SetRemote(remote *udpAddr) {
	i.remote = i.AddRemote(remote)
}

func (i *HostInfo) unlockedBlockRemote(remote *udpAddr) {
	if !i.unlockedIsBadRemote(remote) {
		// We copy here because we are taking something else's memory and we can't trust everything
		i.badRemotes = append(i.badRemotes, remote.Copy())
	}

	for k, v := range i.Remotes {
		if v.Equals(remote) {
			i.Remotes[k] = i.Remotes[len(i.Remotes)-1]
			i.Remotes = i.Remotes[:len(i.Remotes)-1]
			return
		}
	}
}

func (i *HostInfo) unlockedIsBadRemote(remote *udpAddr) bool {
	for _, v := range i.badRemotes {
		if v.Equals(remote) {
			return true
		}
	}
	return false
}

func (i *HostInfo) ClearRemotes() {
	i.remote = nil
	i.Remotes = []*udpAddr{}
}

func (i *HostInfo) ClearConnectionState() {
	i.ConnectionState = nil
}

func (i *HostInfo) RecvErrorExceeded() bool {
	if i.recvError < 3 {
		i.recvError += 1
		return false
	}
	return true
}

func (i *HostInfo) CreateRemoteCIDR(c *cert.NebulaCertificate) {
	if len(c.Details.Ips) == 1 && len(c.Details.Subnets) == 0 {
		// Simple case, no CIDRTree needed
		return
	}

	remoteCidr := NewCIDRTree()
	for _, ip := range c.Details.Ips {
		remoteCidr.AddCIDR(&net.IPNet{IP: ip.IP, Mask: net.IPMask{255, 255, 255, 255}}, struct{}{})
	}

	for _, n := range c.Details.Subnets {
		remoteCidr.AddCIDR(n, struct{}{})
	}
	i.remoteCidr = remoteCidr
}

func (i *HostInfo) logger(l *logrus.Logger) *logrus.Entry {
	if i == nil {
		return logrus.NewEntry(l)
	}

	li := l.WithField("vpnIp", IntIp(i.hostId))

	if connState := i.ConnectionState; connState != nil {
		if peerCert := connState.peerCert; peerCert != nil {
			li = li.WithField("certName", peerCert.Details.Names)
		}
	}

	return li
}

//########################

/*

func (hm *HostMap) DebugRemotes(vpnIp uint32) string {
	s := "\n"
	for _, h := range hm.Hosts {
		for _, r := range h.Remotes {
			s += fmt.Sprintf("%s : %d ## %v\n", r.addr.IP.String(), r.addr.Port, r.probes)
		}
	}
	return s
}

func (i *HostInfo) HandleReply(addr *net.UDPAddr, counter int) {
	for _, r := range i.Remotes {
		if r.addr.IP.Equal(addr.IP) && r.addr.Port == addr.Port {
			r.ProbeReceived(counter)
		}
	}
}

func (i *HostInfo) Probes() []*Probe {
	p := []*Probe{}
	for _, d := range i.Remotes {
		p = append(p, &Probe{Addr: d.addr, Counter: d.Probe()})
	}
	return p
}

*/

// Utility functions

func localIps(l *logrus.Logger, allowList *AllowList) *[]net.IP {
	//FIXME: This function is pretty garbage
	var ips []net.IP
	ifaces, _ := net.Interfaces()
	for _, i := range ifaces {
		allow := allowList.AllowName(i.Name)
		if l.Level >= logrus.TraceLevel {
			l.WithField("interfaceName", i.Name).WithField("allow", allow).Trace("localAllowList.AllowName")
		}

		if !allow {
			continue
		}
		addrs, _ := i.Addrs()
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				//continue
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			//TODO: Filtering out link local for now, this is probably the most correct thing
			//TODO: Would be nice to filter out SLAAC MAC based ips as well
			if ip.IsLoopback() == false && !ip.IsLinkLocalUnicast() {
				allow := allowList.Allow(ip)
				if l.Level >= logrus.TraceLevel {
					l.WithField("localIp", ip).WithField("allow", allow).Trace("localAllowList.Allow")
				}
				if !allow {
					continue
				}

				ips = append(ips, ip)
			}
		}
	}
	return &ips
}

func PrivateIP(ip net.IP) bool {
	//TODO: Private for ipv6 or just let it ride?
	private := false
	_, private24BitBlock, _ := net.ParseCIDR("10.0.0.0/8")
	_, private20BitBlock, _ := net.ParseCIDR("172.16.0.0/12")
	_, private16BitBlock, _ := net.ParseCIDR("192.168.0.0/16")
	private = private24BitBlock.Contains(ip) || private20BitBlock.Contains(ip) || private16BitBlock.Contains(ip)
	return private
}
