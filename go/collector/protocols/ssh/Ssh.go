package ssh

import (
	"bytes"
	"github.com/saichler/l8pollaris/go/pollaris"
	"github.com/saichler/l8pollaris/go/types"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8utils/go/utils/queues"
	strings2 "github.com/saichler/l8utils/go/utils/strings"
	ssh2 "golang.org/x/crypto/ssh"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"
)

var CR = []byte("\n")

type SshCollector struct {
	resources ifs.IResources
	config    *types.Connection
	client    *ssh2.Client
	session   *ssh2.Session
	in        io.WriteCloser
	out       io.Reader
	queue     *queues.Queue
	running   bool
	connected bool
	mtx       *sync.Mutex
}

func (sshc *SshCollector) Protocol() types.Protocol {
	return types.Protocol_PSSH
}

func (sshc *SshCollector) Init(conf *types.Connection, resources ifs.IResources) error {
	sshc.config = conf
	sshc.resources = resources
	sshc.queue = queues.NewQueue("SSh Collector", 1024)
	sshc.running = true
	if conf.Prompt == nil || len(conf.Prompt) == 0 {
		conf.Prompt = make([]string, 1)
		conf.Prompt[0] = "#"
	}
	sshc.mtx = &sync.Mutex{}
	return nil
}

func (sshc *SshCollector) run() {
	for sshc.running {
		buff := make([]byte, 512)
		readBytes, err := sshc.out.Read(buff)
		if err != nil {
			if err.Error() != "EOF" {
				sshc.resources.Logger().Error(err)
			}
			break
		}
		if readBytes > 0 {
			sshc.queue.Add(buff[0:readBytes])
		}
	}
	sshc.resources.Logger().Info("Ssh Collector for host:" + sshc.config.Addr + " is closed.")
}

func (sshc *SshCollector) Connect() error {
	sshconfig := &ssh2.ClientConfig{}
	sshconfig.Timeout = time.Second * time.Duration(sshc.config.Timeout)
	sshconfig.Config = ssh2.Config{}
	sshconfig.User = sshc.config.Username
	pass := ssh2.Password(sshc.config.Password)
	sshconfig.Auth = make([]ssh2.AuthMethod, 1)
	sshconfig.Auth[0] = pass
	sshconfig.HostKeyCallback = ssh2.InsecureIgnoreHostKey()

	hostport := strings2.New(sshc.config.Addr, "/", strconv.Itoa(int(sshc.config.Port))).String()
	client, err := ssh2.Dial("tcp", sshc.config.Addr+":"+strconv.Itoa(int(sshc.config.Port)), sshconfig)
	if err != nil {
		return sshc.resources.Logger().Error("Ssh Dial Error Host:", hostport, err.Error())
	}
	sshc.client = client
	session, err := client.NewSession()
	if err != nil {
		return sshc.resources.Logger().Error("Ssh Session Error Host:", hostport, err.Error())
	}
	sshc.session = session

	if sshc.config.Terminal == "vt100" {
		terminalModes := make(map[uint8]uint32)
		terminalModes[ssh2.TTY_OP_ISPEED] = 38400
		terminalModes[ssh2.TTY_OP_OSPEED] = 38400
		terminalModes[ssh2.ECHO] = 0
		terminalModes[ssh2.OCRNL] = 0
		err = session.RequestPty("vt100", 0, 2048, terminalModes)
		if err != nil {
			return sshc.resources.Logger().Error("Ssh terminal vt100 Error Host:", hostport, err.Error())
		}
	}

	in, _ := session.StdinPipe()
	out, _ := session.StdoutPipe()

	sshc.in = in
	sshc.out = out
	err = session.Shell()

	if err != nil {
		return sshc.resources.Logger().Error("Ssh Shell Error Host:", hostport, err.Error())
	}

	if sshc.config.TerminalCommands != nil {
		for _, cmd := range sshc.config.TerminalCommands {
			time.Sleep(time.Second / 4)
			sshc.in.Write([]byte(cmd))
		}
	}

	go sshc.run()

	time.Sleep(time.Second)

	//Flush welcome message & initial prompt
	/*
		data, err := sshc.exec("", 10)
		if err != nil {
			return errors.New(Join("Ssh Read Error Host:", hostport, err.Error()))
		}*/

	//sshc.setInitialPrompt("#")

	sshc.connected = true

	return nil
}

func (sshc *SshCollector) Disconnect() error {
	sshc.running = false
	if sshc.in != nil {
		sshc.in.Close()
	}
	if sshc.session != nil {
		sshc.session.Close()
	}
	if sshc.client != nil {
		sshc.client.Close()
	}
	sshc.session = nil
	sshc.client = nil
	sshc.connected = false
	return nil
}

func (sshc *SshCollector) setInitialPrompt(str string) {
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
		sshc.resources.Logger().Info("Setting Prompt to:" + prompt)
		sshc.config.Prompt[0] = prompt
	}
}

func (sshc *SshCollector) hasPrompt(data string, count int) bool {
	l := len(sshc.config.Prompt)
	if l == 1 {
		c := strings.Count(data, sshc.config.Prompt[0])
		if c >= count {
			return true
		}
	} else if l == 2 {
		c1 := strings.Count(data, sshc.config.Prompt[0])
		c2 := strings.Count(data, sshc.config.Prompt[1])
		if c1 >= count || c2 >= count {
			return true
		}
	} else {
		for _, prompt := range sshc.config.Prompt {
			c := strings.Count(data, prompt)
			if c >= count {
				return true
			}
		}
	}
	return false
}

func (sshc *SshCollector) exec(cmd string, timeout int64) (string, error) {
	if !sshc.connected {
		err := sshc.Connect()
		if err != nil {
			return err.Error(), err
		}
	}
	if cmd != "" {
		sshc.queue.Clear()
		_, err := sshc.in.Write([]byte(cmd))
		if err != nil {
			return strings2.New("Ssh Write Error Host:", sshc.config.Addr, ":", strconv.Itoa(int(sshc.config.Port))).String(), err
		}
		_, err = sshc.in.Write(CR)
		if err != nil {
			return err.Error(), sshc.resources.Logger().Error("Ssh Write Error Host:", sshc.config.Addr, ":", strconv.Itoa(int(sshc.config.Port)), err.Error())
		}
	}

	result := bytes.Buffer{}
	start := time.Now().Unix()

	cycles := 0
	lastCycleSize := 0
	for time.Now().Unix()-start <= int64(timeout) && !sshc.hasPrompt(result.String(), 1) && cycles < 5 {
		for sshc.queue.Size() > 0 {
			data := sshc.queue.Next().([]byte)
			result.Write(data)
		}
		if !sshc.hasPrompt(result.String(), 1) {
			time.Sleep(time.Second / 10)
		}
		if lastCycleSize == result.Len() {
			cycles++
		}
		lastCycleSize = result.Len()
	}

	return result.String(), nil
}

func (sshc *SshCollector) Exec(job *types.Job) {
	poll, err := pollaris.Poll(job.PollarisName, job.JobName, sshc.resources)
	if err != nil {
		sshc.resources.Logger().Error("Ssh:" + err.Error())
		return
	}
	result, e := sshc.exec(poll.What, job.Timeout)
	if e != nil {
		job.Result = nil
		job.Error = e.Error()
		return
	}
	index := strings.Index(result, poll.What) + len(poll.What) + 1
	if index < len(result) {
		result = result[index:]
	}
	result = strings.Trim(result, "\n")
	result = strings.Trim(result, " ")
	result = strings.Trim(result, "\r")
	for _, prompt := range sshc.config.Prompt {
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
