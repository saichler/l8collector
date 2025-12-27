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

// Package ssh provides an SSH protocol collector implementation for the
// L8Collector service. It enables command execution and data collection
// from remote devices via SSH with interactive shell support.
package ssh

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/saichler/l8pollaris/go/pollaris"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8utils/go/utils/queues"
	strings2 "github.com/saichler/l8utils/go/utils/strings"
	ssh2 "golang.org/x/crypto/ssh"
)

// CR represents a carriage return/newline byte sequence for command termination.
var CR = []byte("\n")

// SshCollector implements the ProtocolCollector interface for SSH-based
// command execution. It maintains a persistent interactive shell session
// to the target device and executes commands by writing to stdin and
// reading responses from stdout.
//
// Features:
//   - Password-based authentication
//   - VT100 terminal emulation support
//   - Configurable command prompts for response detection
//   - Background output reader with queue-based buffering
//   - Terminal initialization commands for device-specific setup
//   - Automatic connection management with reconnection support
//
// The collector uses a background goroutine to continuously read from
// the SSH session and queue the output for command response collection.
type SshCollector struct {
	resources ifs.IResources                // Layer8 resources for logging and security
	config    *l8tpollaris.L8PHostProtocol  // Host configuration with connection details
	client    *ssh2.Client                  // SSH client connection
	session   *ssh2.Session                 // SSH session for shell interaction
	in        io.WriteCloser                // Stdin pipe for command input
	out       io.Reader                     // Stdout pipe for response output
	queue     *queues.Queue                 // Queue for buffering async output reads
	running   bool                          // Flag indicating if background reader is active
	connected bool                          // Connection state flag
	pollOnce  bool                          // Flag indicating at least one poll was attempted
	mtx       *sync.Mutex                   // Mutex for thread-safe operations
}

// Protocol returns the protocol type identifier for SSH.
// This is used by the collector service to route jobs to the correct collector.
func (this *SshCollector) Protocol() l8tpollaris.L8PProtocol {
	return l8tpollaris.L8PProtocol_L8PSSH
}

// Init initializes the SSH collector with the provided host configuration.
// It sets up the output queue and default prompt pattern if none is specified.
// The default prompt is "#" which works for most Unix/Linux systems.
//
// Parameters:
//   - conf: Host protocol configuration containing address, port, and prompts
//   - resources: Layer8 resources for accessing security credentials and logging
//
// Returns:
//   - Always returns nil (initialization cannot fail)
func (this *SshCollector) Init(conf *l8tpollaris.L8PHostProtocol, resources ifs.IResources) error {
	this.config = conf
	this.resources = resources
	this.queue = queues.NewQueue("SSh Collector", 1024)
	this.running = true
	if conf.Prompt == nil || len(conf.Prompt) == 0 {
		conf.Prompt = make([]string, 1)
		conf.Prompt[0] = "#"
	}
	this.mtx = &sync.Mutex{}
	return nil
}

// run is the background goroutine that continuously reads from the SSH
// stdout pipe and queues the data for processing by exec(). It reads in
// 512-byte chunks and runs until the running flag is set to false or
// an EOF is encountered.
func (this *SshCollector) run() {
	for this.running {
		buff := make([]byte, 512)
		readBytes, err := this.out.Read(buff)
		if err != nil {
			if err.Error() != "EOF" {
				this.resources.Logger().Error(err)
			}
			break
		}
		if readBytes > 0 {
			this.queue.Add(buff[0:readBytes])
		}
	}
	this.resources.Logger().Debug(strings2.New("Ssh Collector for host:", this.config.Addr, " is closed.").String())
}

// Connect establishes the SSH connection to the target device.
// It configures the SSH client with password authentication and optionally
// sets up VT100 terminal emulation. After establishing the session, it
// starts the background output reader goroutine.
//
// The connection process:
//  1. Retrieves credentials from the security service
//  2. Establishes TCP connection to the target
//  3. Creates an SSH session
//  4. Optionally configures VT100 terminal mode
//  5. Executes any configured terminal initialization commands
//  6. Starts the background output reader
//
// Returns:
//   - error if any step of the connection process fails
func (this *SshCollector) Connect() error {
	sshconfig := &ssh2.ClientConfig{}
	sshconfig.Timeout = time.Second * time.Duration(this.config.Timeout)
	sshconfig.Config = ssh2.Config{}
	_, user, password, _, err := this.resources.Security().Credential(this.config.CredId, "ssh", this.resources)
	if err != nil {
		panic(err)
	}
	sshconfig.User = user
	pass := ssh2.Password(password)
	sshconfig.Auth = make([]ssh2.AuthMethod, 1)
	sshconfig.Auth[0] = pass
	sshconfig.HostKeyCallback = ssh2.InsecureIgnoreHostKey()

	hostport := strings2.New(this.config.Addr, "/", int(this.config.Port)).String()
	client, err := ssh2.Dial("tcp", strings2.New(this.config.Addr, ":", int(this.config.Port)).String(), sshconfig)
	if err != nil {
		return this.resources.Logger().Error("Ssh Dial Error Host:", hostport, err.Error())
	}
	this.client = client
	session, err := client.NewSession()
	if err != nil {
		return this.resources.Logger().Error("Ssh Session Error Host:", hostport, err.Error())
	}
	this.session = session

	if this.config.Terminal == "vt100" {
		terminalModes := make(map[uint8]uint32)
		terminalModes[ssh2.TTY_OP_ISPEED] = 38400
		terminalModes[ssh2.TTY_OP_OSPEED] = 38400
		terminalModes[ssh2.ECHO] = 0
		terminalModes[ssh2.OCRNL] = 0
		err = session.RequestPty("vt100", 0, 2048, terminalModes)
		if err != nil {
			return this.resources.Logger().Error("Ssh terminal vt100 Error Host:", hostport, err.Error())
		}
	}

	in, _ := session.StdinPipe()
	out, _ := session.StdoutPipe()

	this.in = in
	this.out = out
	err = session.Shell()

	if err != nil {
		return this.resources.Logger().Error("Ssh Shell Error Host:", hostport, err.Error())
	}

	if this.config.TerminalCommands != nil {
		for _, cmd := range this.config.TerminalCommands {
			time.Sleep(time.Second / 4)
			this.in.Write([]byte(cmd))
		}
	}

	go this.run()

	time.Sleep(time.Second)

	//Flush welcome message & initial prompt
	/*
		data, err := this.exec("", 10)
		if err != nil {
			return errors.New(Join("Ssh Read Error Host:", hostport, err.Error()))
		}*/

	//this.setInitialPrompt("#")

	this.connected = true

	return nil
}

// Disconnect closes the SSH connection and releases all resources.
// It stops the background reader, closes the stdin/stdout pipes,
// terminates the session, and closes the client connection.
//
// Returns:
//   - Always returns nil (cleanup is best-effort)
func (this *SshCollector) Disconnect() error {
	this.running = false
	if this.in != nil {
		this.in.Close()
		this.in = nil
	}
	if this.session != nil {
		this.session.Close()
		this.session = nil
	}
	if this.client != nil {
		this.client.Close()
		this.client = nil
	}
	if this.queue != nil {
		this.queue.Shutdown()
		this.queue = nil
	}
	this.connected = false
	return nil
}

// setInitialPrompt extracts the command prompt from the initial connection
// output and updates the prompt configuration. This is used to auto-detect
// the device's actual prompt pattern.
func (this *SshCollector) setInitialPrompt(str string) {
	index := -1
	size := len(str)
	for i := size - 1; i >= 0; i-- {
		if str[i:i+1] == "\n" {
			index = i + 1
			break
		}
	}
	if index != -1 {
		prompt := str[index:]
		this.resources.Logger().Debug(strings2.New("Setting Prompt to:", prompt).String())
		this.config.Prompt[0] = prompt
	}
}

// hasPrompt checks if the accumulated output data contains the expected
// command prompt at least 'count' times. Supports multiple prompt patterns
// configured in the host configuration. Returns true when the prompt is
// detected, indicating the command has completed.
func (this *SshCollector) hasPrompt(data string, count int) bool {
	l := len(this.config.Prompt)
	if l == 1 {
		c := strings.Count(data, this.config.Prompt[0])
		if c >= count {
			return true
		}
	} else if l == 2 {
		c1 := strings.Count(data, this.config.Prompt[0])
		c2 := strings.Count(data, this.config.Prompt[1])
		if c1 >= count || c2 >= count {
			return true
		}
	} else {
		for _, prompt := range this.config.Prompt {
			c := strings.Count(data, prompt)
			if c >= count {
				return true
			}
		}
	}
	return false
}

// exec is the internal command execution method. It sends a command to the
// SSH session and waits for the response until the prompt is detected or
// timeout occurs. The method automatically establishes a connection if needed.
//
// Parameters:
//   - cmd: The command to execute (empty string for reading initial output)
//   - timeout: Maximum time in seconds to wait for the response
//
// Returns:
//   - The command output as a string
//   - error if connection or command execution fails
func (this *SshCollector) exec(cmd string, timeout int64) (string, error) {
	this.pollOnce = true
	if !this.connected {
		err := this.Connect()
		if err != nil {
			return err.Error(), err
		}
	}
	if cmd != "" {
		this.queue.Clear()
		_, err := this.in.Write([]byte(cmd))
		if err != nil {
			return strings2.New("Ssh Write Error Host:", this.config.Addr, ":", int(this.config.Port)).String(), err
		}
		_, err = this.in.Write(CR)
		if err != nil {
			return err.Error(), this.resources.Logger().Error("Ssh Write Error Host:", this.config.Addr, ":", int(this.config.Port), err.Error())
		}
	}

	result := bytes.Buffer{}
	start := time.Now().Unix()

	cycles := 0
	lastCycleSize := 0
	for time.Now().Unix()-start <= int64(timeout) && !this.hasPrompt(result.String(), 1) && cycles < 5 {
		for this.queue.Size() > 0 {
			data := this.queue.Next().([]byte)
			result.Write(data)
		}
		if !this.hasPrompt(result.String(), 1) {
			time.Sleep(time.Second / 10)
		}
		if lastCycleSize == result.Len() {
			cycles++
		}
		lastCycleSize = result.Len()
	}

	return result.String(), nil
}

// Exec executes an SSH command job against the target device.
// The command is obtained from the pollaris configuration using the job's
// PollarisName and JobName. The response is cleaned (removing command echo
// and prompt) and stored in the job's Result field.
//
// Response processing:
//  1. Strips the echoed command from the output
//  2. Removes leading/trailing whitespace and newlines
//  3. Removes the trailing prompt from the output
//  4. Serializes the cleaned result
//
// Parameters:
//   - job: The collection job containing pollaris reference and result storage
func (this *SshCollector) Exec(job *l8tpollaris.CJob) {
	poll, err := pollaris.Poll(job.PollarisName, job.JobName, this.resources)
	if err != nil {
		this.resources.Logger().Error(strings2.New("Ssh:", err.Error()).String())
		return
	}
	result, e := this.exec(poll.What, job.Timeout)
	if e != nil {
		job.Result = nil
		job.Error = e.Error()
		job.ErrorCount++
		return
	} else {
		job.ErrorCount = 0
	}
	index := strings.Index(result, poll.What) + len(poll.What) + 1
	if index < len(result) {
		result = result[index:]
	}
	result = strings.Trim(result, "\n")
	result = strings.Trim(result, " ")
	result = strings.Trim(result, "\r")
	for _, prompt := range this.config.Prompt {
		index = strings.Index(result, prompt)
		if index != -1 {
			result = result[0:index]
			break
		}
	}
	enc := object.NewEncode()
	enc.Add(result)
	job.Result = enc.Data()
}

// Online returns the connection status of the SSH collector.
// Returns true if connected, or if no poll has been attempted yet
// (optimistic status before first poll attempt).
func (this *SshCollector) Online() bool {
	return this.connected || !this.pollOnce
}
