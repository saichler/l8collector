package k8s

import (
	"encoding/base64"
	"github.com/google/uuid"
	"github.com/saichler/l8pollaris/go/pollaris"
	"github.com/saichler/l8pollaris/go/types"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8utils/go/utils/strings"
	"os"
	"os/exec"
)

type Kubernetes struct {
	resources  ifs.IResources
	config     *types.Connection
	kubeConfig string
}

func (this *Kubernetes) Init(config *types.Connection, resources ifs.IResources) error {
	this.resources = resources
	this.config = config
	this.kubeConfig = ".kubeadm-" + config.KukeContext
	data, err := base64.StdEncoding.DecodeString(this.config.KubeConfig)
	if err != nil {
		return err
	}
	err = os.WriteFile(this.kubeConfig, data, 0644)
	return err
}

func (this *Kubernetes) Protocol() types.Protocol {
	return types.Protocol_K8s
}

func (this *Kubernetes) Exec(job *types.Job) {
	this.resources.Logger().Info("K8s Job ", job.PollarisName, ":", job.JobName, " started")
	defer this.resources.Logger().Info("K8s Job ", job.PollarisName, ":", job.JobName, " ended")

	poll, err := pollaris.Poll(job.PollarisName, job.JobName, this.resources)
	if err != nil {
		this.resources.Logger().Error("K8s:" + err.Error())
		return
	}

	script := strings.New("kubectl --kubeconfig=")
	script.Add(this.kubeConfig)
	script.Add(" --context=")
	script.Add(this.config.KukeContext)
	script.Add(" ")
	script.Add(poll.What)
	script.Add("\n")

	id := uuid.New().String()
	in := "./" + id + ".sh"
	defer os.Remove(in)
	os.WriteFile(in, script.Bytes(), 0777)
	c := exec.Command("bash", "-c", in, "2>&1")
	o, e := c.Output()
	if e != nil {
		job.Error = e.Error()
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
