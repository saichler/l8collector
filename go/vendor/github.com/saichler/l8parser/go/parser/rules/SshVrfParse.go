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
	"fmt"
	"strings"

	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8types/go/ifs"
	types2 "github.com/saichler/probler/go/types"
)

// SshVrfParse is a parsing rule that transforms SSH "show vrf" command output
// into VrfInstance structures on the NetworkDevice model.
// It supports multiple vendor output formats via the "format" parameter.
type SshVrfParse struct{}

// Name returns the rule identifier "SshVrfParse".
func (this *SshVrfParse) Name() string {
	return "SshVrfParse"
}

// ParamNames returns the required parameter names for this rule.
func (this *SshVrfParse) ParamNames() []string {
	return []string{"format"}
}

// Parse executes the SshVrfParse rule, parsing SSH VRF output into VrfInstance structures.
func (this *SshVrfParse) Parse(resources ifs.IResources, workSpace map[string]interface{}, params map[string]*l8tpollaris.L8PParameter, any interface{}, pollWhat string) error {
	input := workSpace[Input]
	if input == nil {
		return errors.New("SshVrfParse: no input data")
	}

	// Get the raw SSH output as string
	var sshOutput string
	switch v := input.(type) {
	case string:
		sshOutput = v
	case []byte:
		sshOutput = string(v)
	default:
		return errors.New("SshVrfParse: input is not a string: " + fmt.Sprintf("%T", input))
	}

	if strings.TrimSpace(sshOutput) == "" {
		return nil
	}

	// Get format parameter
	formatParam := params["format"]
	if formatParam == nil || formatParam.Value == "" {
		return errors.New("SshVrfParse: missing 'format' parameter")
	}

	networkDevice, ok := any.(*types2.NetworkDevice)
	if !ok {
		return errors.New("SshVrfParse: target is not a NetworkDevice")
	}

	vrfs := parseVrfOutput(sshOutput, formatParam.Value)
	if len(vrfs) == 0 {
		return nil
	}

	// Set VRFs on NetworkDevice
	ensureLogical(networkDevice)
	networkDevice.Logicals["logical-0"].Vrfs = vrfs

	return nil
}

// ensureLogical ensures the NetworkDevice has a logical-0 entry.
func ensureLogical(nd *types2.NetworkDevice) {
	if nd.Logicals == nil {
		nd.Logicals = make(map[string]*types2.Logical)
	}
	if _, exists := nd.Logicals["logical-0"]; !exists {
		nd.Logicals["logical-0"] = &types2.Logical{Id: "logical-0"}
	}
}

// parseVrfOutput dispatches to the appropriate vendor-specific parser.
func parseVrfOutput(output, format string) []*types2.VrfInstance {
	switch format {
	case "iosxr":
		return parseIosXrVrf(output)
	case "ios":
		return parseIosVrf(output)
	case "nxos":
		return parseNxosVrf(output)
	case "junos":
		return parseJunosVrf(output)
	case "timos":
		return parseTimosVrf(output)
	case "vrp":
		return parseVrpVrf(output)
	case "eos":
		return parseEosVrf(output)
	case "voss":
		return parseVossVrf(output)
	case "univerge":
		return parseUniverseVrf(output)
	default:
		return parseGenericVrf(output)
	}
}

// parseIosXrVrf parses Cisco IOS XR "show vrf all detail" output.
func parseIosXrVrf(output string) []*types2.VrfInstance {
	return parseBlockVrf(output, "VRF ", "RD ", "Import RT:", "Export RT:", "  ")
}

// parseIosVrf parses Cisco IOS/XE "show ip vrf detail" output.
func parseIosVrf(output string) []*types2.VrfInstance {
	return parseBlockVrf(output, "VRF ", "RD ", "Import RT:", "Export RT:", "  ")
}

// parseNxosVrf parses Cisco NX-OS "show vrf detail" output.
func parseNxosVrf(output string) []*types2.VrfInstance {
	return parseBlockVrf(output, "VRF-Name:", "RD:", "Import RT:", "Export RT:", "  ")
}

// parseJunosVrf parses Juniper Junos "show route instance detail" output.
func parseJunosVrf(output string) []*types2.VrfInstance {
	vrfs := make([]*types2.VrfInstance, 0)
	lines := strings.Split(output, "\n")
	var current *types2.VrfInstance

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Instance name line (not indented, ends with colon or contains "Instance:")
		if strings.Contains(trimmed, "Instance:") || (!strings.HasPrefix(line, " ") && strings.Contains(trimmed, ":")) {
			if current != nil {
				current.Status = types2.VrfStatus(1) // ACTIVE
				vrfs = append(vrfs, current)
			}
			name := extractValue(trimmed, "Instance:")
			if name == "" {
				name = strings.TrimSuffix(strings.TrimSpace(trimmed), ":")
			}
			current = &types2.VrfInstance{VrfName: name}
		}

		if current == nil {
			continue
		}

		if strings.Contains(trimmed, "Route-distinguisher:") || strings.Contains(trimmed, "RD:") {
			current.RouteDistinguisher = extractValue(trimmed, "Route-distinguisher:")
			if current.RouteDistinguisher == "" {
				current.RouteDistinguisher = extractValue(trimmed, "RD:")
			}
		}
		if strings.Contains(trimmed, "Import:") {
			rt := extractValue(trimmed, "Import:")
			if rt != "" {
				current.RouteTargetsImport = append(current.RouteTargetsImport, rt)
			}
		}
		if strings.Contains(trimmed, "Export:") {
			rt := extractValue(trimmed, "Export:")
			if rt != "" {
				current.RouteTargetsExport = append(current.RouteTargetsExport, rt)
			}
		}
		if strings.Contains(trimmed, "Interfaces:") {
			ifaces := extractValue(trimmed, "Interfaces:")
			parseInterfaceList(current, ifaces)
		}
	}

	if current != nil {
		current.Status = types2.VrfStatus(1)
		vrfs = append(vrfs, current)
	}
	return vrfs
}

// parseTimosVrf parses Nokia TiMOS "show router vrf" / "show service vprn" output.
func parseTimosVrf(output string) []*types2.VrfInstance {
	return parseBlockVrf(output, "VPRN ", "RD:", "Import:", "Export:", "  ")
}

// parseVrpVrf parses Huawei VRP "display ip vpn-instance verbose" output.
func parseVrpVrf(output string) []*types2.VrfInstance {
	return parseBlockVrf(output, "VPN-Instance :", "Route Distinguisher :", "Import VPN Targets :", "Export VPN Targets :", "  ")
}

// parseEosVrf parses Arista EOS "show vrf detail" output.
func parseEosVrf(output string) []*types2.VrfInstance {
	return parseBlockVrf(output, "VRF ", "RD:", "Import RT:", "Export RT:", "  ")
}

// parseVossVrf parses Extreme VOSS "show ip vrf detail" output.
func parseVossVrf(output string) []*types2.VrfInstance {
	return parseBlockVrf(output, "VRF ", "RD:", "Import RT:", "Export RT:", "  ")
}

// parseUniverseVrf parses NEC UNIVERGE "show ip vrf detail" output.
func parseUniverseVrf(output string) []*types2.VrfInstance {
	return parseBlockVrf(output, "VRF ", "RD ", "Import RT:", "Export RT:", "  ")
}

// parseGenericVrf is a fallback parser for unknown formats.
func parseGenericVrf(output string) []*types2.VrfInstance {
	return parseBlockVrf(output, "VRF ", "RD", "Import", "Export", "  ")
}

// parseBlockVrf is a generic block-based VRF parser that works for most vendor formats.
// It splits the output into VRF blocks identified by nameMarker and extracts RD and RT values.
func parseBlockVrf(output, nameMarker, rdMarker, importMarker, exportMarker, interfaceIndent string) []*types2.VrfInstance {
	vrfs := make([]*types2.VrfInstance, 0)
	lines := strings.Split(output, "\n")
	var current *types2.VrfInstance

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Detect VRF name line
		if strings.Contains(trimmed, nameMarker) {
			if current != nil {
				current.Status = types2.VrfStatus(1) // ACTIVE
				vrfs = append(vrfs, current)
			}
			name := extractValue(trimmed, nameMarker)
			// Clean common suffixes
			name = strings.TrimSuffix(name, ";")
			name = strings.TrimSuffix(name, ",")
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			current = &types2.VrfInstance{VrfName: name}
			continue
		}

		if current == nil {
			continue
		}

		if strings.Contains(trimmed, rdMarker) {
			rd := extractValue(trimmed, rdMarker)
			rd = strings.TrimSuffix(rd, ";")
			current.RouteDistinguisher = strings.TrimSpace(rd)
		}
		if strings.Contains(trimmed, importMarker) {
			rt := extractValue(trimmed, importMarker)
			rt = strings.TrimSpace(strings.TrimSuffix(rt, ";"))
			if rt != "" {
				for _, r := range strings.Split(rt, ",") {
					r = strings.TrimSpace(r)
					if r != "" {
						current.RouteTargetsImport = append(current.RouteTargetsImport, r)
					}
				}
			}
		}
		if strings.Contains(trimmed, exportMarker) {
			rt := extractValue(trimmed, exportMarker)
			rt = strings.TrimSpace(strings.TrimSuffix(rt, ";"))
			if rt != "" {
				for _, r := range strings.Split(rt, ",") {
					r = strings.TrimSpace(r)
					if r != "" {
						current.RouteTargetsExport = append(current.RouteTargetsExport, r)
					}
				}
			}
		}
		if strings.Contains(trimmed, "Interfaces:") || strings.Contains(trimmed, "Interface:") || strings.Contains(trimmed, "interfaces:") {
			ifaces := extractValue(trimmed, "Interfaces:")
			if ifaces == "" {
				ifaces = extractValue(trimmed, "Interface:")
			}
			if ifaces == "" {
				ifaces = extractValue(trimmed, "interfaces:")
			}
			parseInterfaceList(current, ifaces)
		}
	}

	if current != nil {
		current.Status = types2.VrfStatus(1)
		vrfs = append(vrfs, current)
	}
	return vrfs
}

// extractValue extracts the value after a marker in a line.
func extractValue(line, marker string) string {
	idx := strings.Index(line, marker)
	if idx < 0 {
		return ""
	}
	return strings.TrimSpace(line[idx+len(marker):])
}

// parseInterfaceList parses a comma/space separated interface list into InterfaceIds.
func parseInterfaceList(vrf *types2.VrfInstance, ifaces string) {
	if ifaces == "" {
		return
	}
	// Split on comma or whitespace
	parts := strings.FieldsFunc(ifaces, func(r rune) bool {
		return r == ',' || r == ' '
	})
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			vrf.InterfaceIds = append(vrf.InterfaceIds, p)
		}
	}
}
