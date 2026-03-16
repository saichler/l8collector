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
	"strconv"
	"strings"

	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8reflect/go/reflect/properties"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
)

// SshNvidiaSmiParse is a parsing rule that transforms SSH command output from
// nvidia-smi subcommands, show version, and lscpu into GpuDevice properties.
// The "format" parameter selects which parser to use.
type SshNvidiaSmiParse struct{}

// Name returns the rule identifier "SshNvidiaSmiParse".
func (this *SshNvidiaSmiParse) Name() string {
	return "SshNvidiaSmiParse"
}

// ParamNames returns the required parameter names for this rule.
func (this *SshNvidiaSmiParse) ParamNames() []string {
	return []string{"format"}
}

// Parse executes the SshNvidiaSmiParse rule, dispatching to the appropriate format parser.
func (this *SshNvidiaSmiParse) Parse(resources ifs.IResources, workSpace map[string]interface{},
	params map[string]*l8tpollaris.L8PParameter, any interface{}, pollWhat string) error {

	input := workSpace[Input]
	if input == nil {
		return errors.New("SshNvidiaSmiParse: no input data")
	}

	var sshOutput string
	switch v := input.(type) {
	case string:
		sshOutput = v
	case []byte:
		sshOutput = string(v)
	default:
		return errors.New("SshNvidiaSmiParse: input is not a string: " + fmt.Sprintf("%T", input))
	}

	if strings.TrimSpace(sshOutput) == "" {
		resources.Logger().Error("SshNvidiaSmiParse: empty SSH output for ", pollWhat)
		return nil
	}

	formatParam := params["format"]
	if formatParam == nil || formatParam.Value == "" {
		return errors.New("SshNvidiaSmiParse: missing 'format' parameter")
	}

	propertyId := ""
	if pid, ok := workSpace[PropertyId]; ok {
		propertyId, _ = pid.(string)
	}

	var stamp int64
	if ended, ok := workSpace[JobEnded]; ok {
		if s, ok := ended.(int64); ok {
			stamp = s
		}
	}

	switch formatParam.Value {
	case "utilization":
		return parseNvidiaSmiUtilization(resources, sshOutput, propertyId, stamp, any)
	case "temperature":
		return parseNvidiaSmiTemperature(resources, sshOutput, propertyId, stamp, any)
	case "power":
		return parseNvidiaSmiPower(resources, sshOutput, propertyId, any)
	case "version":
		return parseShowVersion(resources, sshOutput, propertyId, any)
	case "lscpu":
		return parseLscpu(resources, sshOutput, propertyId, any)
	default:
		return errors.New("SshNvidiaSmiParse: unknown format: " + formatParam.Value)
	}
}

// parseNvidiaSmiUtilization parses "nvidia-smi -q -d UTILIZATION" output.
// Extracts encoder and decoder utilization per GPU.
func parseNvidiaSmiUtilization(resources ifs.IResources, output, propertyId string, stamp int64, any interface{}) error {
	gpuKey := ""
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "GPU ") && strings.Contains(trimmed, ":") {
			gpuKey = extractGpuPciBusId(trimmed)
			if gpuKey != "" {
				setGpuProperty(resources, propertyId, gpuKey, "pcibusid", gpuKey, any)
			}
			continue
		}

		if gpuKey == "" {
			continue
		}

		if strings.HasPrefix(trimmed, "Encoder") && strings.Contains(trimmed, ":") {
			val := extractPercentValue(trimmed)
			if val >= 0 {
				setGpuTimeSeries(resources, propertyId, gpuKey, "encoderutilizationpercent", stamp, val, any)
			}
		}
		if strings.HasPrefix(trimmed, "Decoder") && strings.Contains(trimmed, ":") {
			val := extractPercentValue(trimmed)
			if val >= 0 {
				setGpuTimeSeries(resources, propertyId, gpuKey, "decoderutilizationpercent", stamp, val, any)
			}
		}
	}
	return nil
}

// parseNvidiaSmiTemperature parses "nvidia-smi -q -d TEMPERATURE" output.
// Extracts memory temperature, shutdown temp, and slowdown temp per GPU.
func parseNvidiaSmiTemperature(resources ifs.IResources, output, propertyId string, stamp int64, any interface{}) error {
	gpuKey := ""
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "GPU ") && strings.Contains(trimmed, ":") {
			gpuKey = extractGpuPciBusId(trimmed)
			if gpuKey != "" {
				setGpuProperty(resources, propertyId, gpuKey, "pcibusid", gpuKey, any)
			}
			continue
		}

		if gpuKey == "" {
			continue
		}

		if strings.Contains(trimmed, "GPU Memory Temp") || strings.Contains(trimmed, "Memory Current Temp") {
			val := extractTempValue(trimmed)
			if val >= 0 {
				setGpuTimeSeries(resources, propertyId, gpuKey, "memorytemperaturecelsius", stamp, val, any)
			}
		}
		if strings.Contains(trimmed, "GPU Shutdown Temp") {
			val := extractTempValue(trimmed)
			if val >= 0 {
				setGpuProperty(resources, propertyId, gpuKey, "shutdowntemperature", float64(val), any)
			}
		}
		if strings.Contains(trimmed, "GPU Slowdown Temp") {
			val := extractTempValue(trimmed)
			if val >= 0 {
				setGpuProperty(resources, propertyId, gpuKey, "slowdowntemperature", float64(val), any)
			}
		}
	}
	return nil
}

// parseNvidiaSmiPower parses "nvidia-smi -q -d POWER" output.
// Extracts power limit per GPU.
func parseNvidiaSmiPower(resources ifs.IResources, output, propertyId string, any interface{}) error {
	gpuKey := ""
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "GPU ") && strings.Contains(trimmed, ":") {
			gpuKey = extractGpuPciBusId(trimmed)
			if gpuKey != "" {
				setGpuProperty(resources, propertyId, gpuKey, "pcibusid", gpuKey, any)
			}
			continue
		}

		if gpuKey == "" {
			continue
		}

		if strings.Contains(trimmed, "Default Power Limit") || strings.Contains(trimmed, "Power Limit") {
			if strings.Contains(trimmed, "Enforced") || strings.Contains(trimmed, "Min") || strings.Contains(trimmed, "Max") {
				continue
			}
			val := extractWattValue(trimmed)
			if val >= 0 {
				setGpuProperty(resources, propertyId, gpuKey, "powerlimitwatts", val, any)
			}
		}
	}
	return nil
}

// parseShowVersion parses "show version" output for kernel version, model, and serial number.
func parseShowVersion(resources ifs.IResources, output, propertyId string, any interface{}) error {
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		lower := strings.ToLower(trimmed)

		if strings.Contains(lower, "kernel") && strings.Contains(trimmed, ":") {
			val := extractKV(trimmed)
			if val != "" {
				setProperty(resources, propertyId+".kernelversion", val, any)
			}
		}

		if (strings.Contains(lower, "dgx") || strings.Contains(lower, "hgx")) && strings.Contains(lower, "software") {
			val := extractKV(trimmed)
			if val != "" {
				setProperty(resources, propertyId+".model", val, any)
			}
		}

		if strings.Contains(lower, "serial") && strings.Contains(trimmed, ":") {
			val := extractKV(trimmed)
			if val != "" {
				setProperty(resources, propertyId+".serialnumber", val, any)
			}
		}
	}
	return nil
}

// parseLscpu parses "lscpu" output for CPU sockets and total cores.
func parseLscpu(resources ifs.IResources, output, propertyId string, any interface{}) error {
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if strings.HasPrefix(trimmed, "Socket(s):") {
			val := extractKV(trimmed)
			if n, err := strconv.ParseUint(strings.TrimSpace(val), 10, 32); err == nil {
				setProperty(resources, propertyId+".cpusockets", uint32(n), any)
			}
		}

		if strings.HasPrefix(trimmed, "CPU(s):") && !strings.Contains(trimmed, "On-line") && !strings.Contains(trimmed, "NUMA") {
			val := extractKV(trimmed)
			if n, err := strconv.ParseUint(strings.TrimSpace(val), 10, 32); err == nil {
				setProperty(resources, propertyId+".cpucorestotal", uint32(n), any)
			}
		}
	}
	return nil
}

// --- Helper functions ---

// extractGpuPciBusId extracts the PCI Bus ID from a line like "GPU 00000000:07:00.0".
// Returns empty string if the line doesn't contain a valid PCI Bus ID.
// PCI Bus ID format: hex digits, colons, and dots (e.g., "00000000:07:00.0").
func extractGpuPciBusId(line string) string {
	trimmed := strings.TrimSpace(line)
	rest := strings.TrimPrefix(trimmed, "GPU ")
	rest = strings.TrimSpace(rest)
	// Validate PCI Bus ID format: must contain hex digits, colons, and dots only
	for _, c := range rest {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') || c == ':' || c == '.') {
			return ""
		}
	}
	if len(rest) == 0 {
		return ""
	}
	return rest
}

// extractPercentValue extracts a percentage value from a line like "Encoder : 45 %".
func extractPercentValue(line string) float64 {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return -1
	}
	valStr := strings.TrimSpace(parts[1])
	valStr = strings.TrimSuffix(valStr, "%")
	valStr = strings.TrimSpace(valStr)
	if val, err := strconv.ParseFloat(valStr, 64); err == nil {
		return val
	}
	return -1
}

// extractTempValue extracts a temperature value from a line like "GPU Shutdown Temp : 92 C".
func extractTempValue(line string) float64 {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return -1
	}
	valStr := strings.TrimSpace(parts[1])
	valStr = strings.TrimSuffix(valStr, "C")
	valStr = strings.TrimSpace(valStr)
	if val, err := strconv.ParseFloat(valStr, 64); err == nil {
		return val
	}
	return -1
}

// extractWattValue extracts a watt value from a line like "Default Power Limit : 400.00 W".
func extractWattValue(line string) float64 {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return -1
	}
	valStr := strings.TrimSpace(parts[1])
	valStr = strings.TrimSuffix(valStr, "W")
	valStr = strings.TrimSpace(valStr)
	if val, err := strconv.ParseFloat(valStr, 64); err == nil {
		return val
	}
	return -1
}

// extractKV extracts the value part of a "Key : Value" line.
func extractKV(line string) string {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

// setGpuTimeSeries sets a time series value on a per-GPU property.
func setGpuTimeSeries(resources ifs.IResources, propertyId string, gpuKey string, field string, stamp int64, value float64, any interface{}) {
	fullId := fmt.Sprintf("%s<{24}%s>.%s", propertyId, gpuKey, field)
	point := &l8api.L8TimeSeriesPoint{Stamp: stamp, Value: value}
	instance, err := properties.PropertyOf(fullId, resources)
	if err != nil {
		resources.Logger().Error("setGpuTimeSeries: PropertyOf failed for '", fullId, "': ", err.Error())
		return
	}
	if instance == nil {
		resources.Logger().Error("setGpuTimeSeries: PropertyOf returned nil for '", fullId, "'")
		return
	}
	instance.Set(any, point)
}

// setGpuProperty sets a static value on a per-GPU property.
func setGpuProperty(resources ifs.IResources, propertyId string, gpuKey string, field string, value interface{}, any interface{}) {
	fullId := fmt.Sprintf("%s<{24}%s>.%s", propertyId, gpuKey, field)
	instance, err := properties.PropertyOf(fullId, resources)
	if err != nil {
		resources.Logger().Error("setGpuProperty: PropertyOf failed for '", fullId, "': ", err.Error())
		return
	}
	if instance == nil {
		resources.Logger().Error("setGpuProperty: PropertyOf returned nil for '", fullId, "'")
		return
	}
	instance.Set(any, value)
}

// setProperty sets a value on a direct property path.
func setProperty(resources ifs.IResources, propertyId string, value interface{}, any interface{}) {
	modifiedId := injectIndexOrKey(propertyId, nil)
	instance, err := properties.PropertyOf(modifiedId, resources)
	if err != nil {
		resources.Logger().Error("setProperty: PropertyOf failed for '", modifiedId, "': ", err.Error())
		return
	}
	if instance == nil {
		resources.Logger().Error("setProperty: PropertyOf returned nil for '", modifiedId, "'")
		return
	}
	instance.Set(any, value)
}
