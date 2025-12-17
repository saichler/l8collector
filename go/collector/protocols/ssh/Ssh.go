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

var CR = []byte("\n")

type SshCollector struct {
	resources ifs.IResources
	config    *l8tpollaris.L8PHostProtocol
	client    *ssh2.Client
	session   *ssh2.Session
	in        io.WriteCloser
	out       io.Reader
	queue     *queues.Queue
	running   bool
	connected bool
	pollOnce  bool
	mtx       *sync.Mutex
}

func (this *SshCollector) Protocol() l8tpollaris.L8PProtocol {
	return l8tpollaris.L8PProtocol_L8PSSH
}

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
	this.resources.Logger().Info(strings2.New("Ssh Collector for host:", this.config.Addr, " is closed.").String())
}

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
		this.resources.Logger().Info(strings2.New("Setting Prompt to:", prompt).String())
		this.config.Prompt[0] = prompt
	}
}

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

func (this *SshCollector) Online() bool {
	return this.connected || !this.pollOnce
}
