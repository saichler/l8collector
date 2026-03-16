package aggregator

import (
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8utils/go/utils/queues"
	"reflect"
	"time"
)

type Aggregator struct {
	vnic              ifs.IVNic
	queue             *queues.Queue
	running           bool
	intervalInSeconds int64
	timeoutInSeconds  int64
}

func NewAggregator(vnic ifs.IVNic, intervalInSeconds, timeoutInSeconds int64) *Aggregator {
	agg := &Aggregator{}
	agg.vnic = vnic
	agg.queue = queues.NewQueue("Aggregator", 100000)
	agg.running = true
	agg.intervalInSeconds = intervalInSeconds

	go agg.start()
	return agg
}

type ElemEntry struct {
	any         interface{}
	destination string
	serviceName string
	serviceArea byte
	action      ifs.Action
	method      ifs.VNicMethod
}

func (this *Aggregator) Shutdown() {
	this.running = false
	this.queue.Shutdown()
}

func (this *Aggregator) AddElement(any interface{}, method ifs.VNicMethod, destination, serviceName string, serviceArea byte, action ifs.Action) {
	entry := &ElemEntry{any: any, serviceName: serviceName, serviceArea: serviceArea, action: action, method: method, destination: destination}
	this.queue.Add(entry)
}

func (this *Aggregator) start() {
	for this.running {
		time.Sleep(time.Second * time.Duration(this.intervalInSeconds))
		this.flush()
	}
}

func (this *Aggregator) flush() {
	entries := this.queue.Clear()
	var method ifs.VNicMethod
	destination := ""
	serviceName := ""
	serviceArea := byte(0)
	var action ifs.Action

	buff := make([]interface{}, 0)
	for _, en := range entries {
		entry := en.(*ElemEntry)
		if method != entry.method || serviceName != entry.serviceName ||
			serviceArea != entry.serviceArea || action != entry.action ||
			destination != entry.destination {
			this.send(method, destination, serviceName, serviceArea, action, buff)
			buff = make([]interface{}, 0)
		}
		v := reflect.ValueOf(entry.any)
		if v.Kind() == reflect.Slice {
			for i := 0; i < v.Len(); i++ {
				buff = append(buff, v.Index(i).Interface())
			}
		} else {
			buff = append(buff, entry.any)
		}
		method = entry.method
		destination = entry.destination
		serviceName = entry.serviceName
		serviceArea = entry.serviceArea
		action = entry.action
	}
	this.send(method, destination, serviceName, serviceArea, action, buff)
}

func (this *Aggregator) send(method ifs.VNicMethod, destination, serviceName string, serviceArea byte, action ifs.Action, buff []interface{}) {
	if len(buff) == 0 {
		return
	}
	var err error
	switch method {
	case ifs.Unicast:
		err = this.vnic.Unicast(destination, serviceName, serviceArea, action, buff)
	case ifs.Multicast:
		err = this.vnic.Multicast(serviceName, serviceArea, action, buff)
	case ifs.RoundRobin:
		err = this.vnic.RoundRobin(serviceName, serviceArea, action, buff)
	case ifs.RoundRobinRequest:
		resp := this.vnic.RoundRobinRequest(serviceName, serviceArea, action, buff, int(this.timeoutInSeconds))
		if resp != nil && resp.Error() != nil {
			err = resp.Error()
		}
	case ifs.Proximity:
		err = this.vnic.Proximity(serviceName, serviceArea, action, buff)
	case ifs.ProximityRequest:
		resp := this.vnic.ProximityRequest(serviceName, serviceArea, action, buff, int(this.timeoutInSeconds))
		if resp != nil && resp.Error() != nil {
			err = resp.Error()
		}
	case ifs.Leader:
		err = this.vnic.Leader(serviceName, serviceArea, action, buff)
	case ifs.LeaderRequest:
		resp := this.vnic.LeaderRequest(serviceName, serviceArea, action, buff, int(this.timeoutInSeconds))
		if resp != nil && resp.Error() != nil {
			err = resp.Error()
		}
	case ifs.Local:
		err = this.vnic.Local(serviceName, serviceArea, action, buff)
	case ifs.LocalRequest:
		resp := this.vnic.LocalRequest(serviceName, serviceArea, action, buff, int(this.timeoutInSeconds))
		if resp != nil && resp.Error() != nil {
			err = resp.Error()
		}
	case ifs.Request:
		resp := this.vnic.Request(destination, serviceName, serviceArea, action, buff, int(this.timeoutInSeconds))
		if resp != nil && resp.Error() != nil {
			err = resp.Error()
		}
	}
	if err != nil {
		this.vnic.Resources().Logger().Error(err.Error())
	}
}
