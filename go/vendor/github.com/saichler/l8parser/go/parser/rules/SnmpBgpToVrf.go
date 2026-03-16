/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
You may obtain a copy of the License at:

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package rules

import (
	"errors"
	"strings"

	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8types/go/ifs"
	types2 "github.com/saichler/probler/go/types"
)

// SnmpBgpToVrf is a bulk parsing rule that transforms BGP4 MIB (1.3.6.1.2.1.15) walk results
// into a VrfInstance.BgpInfo structure on the NetworkDevice model.
// It extracts global BGP parameters (version, local AS) and the peer table.
type SnmpBgpToVrf struct{}

// Name returns the rule identifier "SnmpBgpToVrf".
func (this *SnmpBgpToVrf) Name() string {
	return "SnmpBgpToVrf"
}

// ParamNames returns the required parameter names for this rule.
func (this *SnmpBgpToVrf) ParamNames() []string {
	return []string{}
}

// Parse executes the SnmpBgpToVrf rule, building BgpInfo from BGP4 MIB walk data.
func (this *SnmpBgpToVrf) Parse(resources ifs.IResources, workSpace map[string]interface{}, params map[string]*l8tpollaris.L8PParameter, any interface{}, pollWhat string) error {
	input := workSpace[Input]
	if input == nil {
		return errors.New("SnmpBgpToVrf: no input data")
	}

	cmap, ok := input.(*l8tpollaris.CMap)
	if !ok {
		return errors.New("SnmpBgpToVrf: input is not a CMap")
	}

	if len(cmap.Data) == 0 {
		return nil // No BGP data available
	}

	networkDevice, ok := any.(*types2.NetworkDevice)
	if !ok {
		return errors.New("SnmpBgpToVrf: target is not a NetworkDevice")
	}

	bgpInfo := &types2.BgpInfo{}

	// Extract global BGP params
	localAs := ospfGetInt64(cmap, ".1.3.6.1.2.1.15.2.0", resources)
	if localAs == 0 {
		return nil // BGP not running on this device
	}

	bgpInfo.BgpEnabled = true
	bgpInfo.AsNumber = uint32(localAs)

	// Extract peers from peer table (15.3.1.*)
	bgpInfo.Peers = bgpExtractPeers(cmap, resources, localAs)

	// Set on NetworkDevice
	ensureLogicalVrf(networkDevice)
	vrf := networkDevice.Logicals["logical-0"].Vrfs[0]
	vrf.BgpInfo = bgpInfo

	return nil
}

// bgpExtractPeers builds BgpPeer entries from the BGP peer table (15.3.1.*).
func bgpExtractPeers(cmap *l8tpollaris.CMap, resources ifs.IResources, localAs int64) []*types2.BgpPeer {
	peers := make([]*types2.BgpPeer, 0)

	// Find all peer IPs from bgpPeerIdentifier (15.3.1.1.<ip>)
	peerIdPrefix := ".1.3.6.1.2.1.15.3.1.1."
	peerStatePrefix := ".1.3.6.1.2.1.15.3.1.2."
	peerRemoteAsPrefix := ".1.3.6.1.2.1.15.3.1.7."
	peerInUpdatesPrefix := ".1.3.6.1.2.1.15.3.1.8."
	peerOutUpdatesPrefix := ".1.3.6.1.2.1.15.3.1.9."
	peerPrefixAcceptedPrefix := ".1.3.6.1.2.1.15.3.1.24."

	for key := range cmap.Data {
		if !strings.HasPrefix(key, peerIdPrefix) {
			continue
		}
		peerIp := strings.TrimPrefix(key, peerIdPrefix) // e.g., "10.1.1.2"

		peer := &types2.BgpPeer{}
		peer.PeerId = ospfGetString(cmap, peerIdPrefix+peerIp, resources)
		peer.PeerIp = peerIp

		remoteAs := ospfGetInt64(cmap, peerRemoteAsPrefix+peerIp, resources)
		peer.PeerAs = uint32(remoteAs)

		// Map SNMP peer state (1-6) to protobuf BgpPeerState (1-6, 0=unknown)
		snmpState := ospfGetInt64(cmap, peerStatePrefix+peerIp, resources)
		if snmpState >= 1 && snmpState <= 6 {
			peer.State = types2.BgpPeerState(snmpState)
		}

		// Determine peer type (iBGP vs eBGP)
		if remoteAs == localAs {
			peer.PeerType = types2.BgpPeerType(1) // IBGP
		} else {
			peer.PeerType = types2.BgpPeerType(2) // EBGP
		}

		peer.RoutesReceived = uint32(ospfGetInt64(cmap, peerPrefixAcceptedPrefix+peerIp, resources))

		// Use inbound/outbound update counts as rough sent estimate
		inUpdates := ospfGetInt64(cmap, peerInUpdatesPrefix+peerIp, resources)
		outUpdates := ospfGetInt64(cmap, peerOutUpdatesPrefix+peerIp, resources)
		_ = inUpdates  // available for future statistics
		_ = outUpdates // available for future statistics

		peers = append(peers, peer)
	}

	return peers
}
