package k8s

import (
	"encoding/base64"
	"os"
	"os/exec"

	"github.com/google/uuid"
	"github.com/saichler/l8collector/go/collector/common"
	"github.com/saichler/l8pollaris/go/pollaris"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8utils/go/utils/strings"
)

type Kubernetes struct {
	resources  ifs.IResources
	config     *l8tpollaris.L8PHostProtocol
	kubeConfig string
	context    string
	connected  bool
}

func (this *Kubernetes) Init(config *l8tpollaris.L8PHostProtocol, resources ifs.IResources) error {
	this.resources = resources
	this.config = config
	_, context, kubeconfig, _, err := this.resources.Security().Credential(this.config.CredId, "kubeconfig", this.resources)
	if err != nil {
		panic(err)
	}
	this.context = context
	this.kubeConfig = ".kubeadm-" + context
	data, err := base64.StdEncoding.DecodeString(kubeconfig)
	if err != nil {
		return err
	}
	err = os.WriteFile(this.kubeConfig, data, 0644)
	return err
}

func (this *Kubernetes) Protocol() l8tpollaris.L8PProtocol {
	return l8tpollaris.L8PProtocol_L8PKubectl
}

func (this *Kubernetes) Exec(job *l8tpollaris.CJob) {
	this.resources.Logger().Info("K8s Job ", job.PollarisName, ":", job.JobName, " started")
	defer this.resources.Logger().Info("K8s Job ", job.PollarisName, ":", job.JobName, " ended")

	poll, err := pollaris.Poll(job.PollarisName, job.JobName, this.resources)
	if err != nil {
		this.resources.Logger().Error(strings.New("K8s:", err.Error()).String())
		return
	}

	script := strings.New("kubectl --kubeconfig=")
	script.Add(this.kubeConfig)
	script.Add(" --context=")
	script.Add(this.context)
	script.Add(" ")
	script.Add(common.ReplaceArguments(poll.What, job))
	script.Add("\n")

	id := uuid.New().String()
	in := strings.New("./", id, ".sh").String()
	defer os.Remove(in)
	os.WriteFile(in, script.Bytes(), 0777)
	c := exec.Command("bash", "-c", in, "2>&1")
	o, e := c.Output()
	if e != nil {
		job.Error = e.Error()
		job.ErrorCount++
	} else {
		job.ErrorCount = 0
	}
	obj := object.NewEncode()
	obj.Add(string(o))
	job.Result = obj.Data()
}

func (this *Kubernetes) Connect() error {
	return nil
}

func (this *Kubernetes) Disconnect() error {
	return nil
}

func (this *Kubernetes) Online() bool {
	return this.connected
}
