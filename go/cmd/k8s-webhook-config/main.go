package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/saichler/l8collector/go/collector/protocols/k8sclient"
)

func main() {
	name := flag.String("name", "l8collector-k8s", "webhook name")
	serviceName := flag.String("service", "l8collector", "kubernetes service name")
	namespace := flag.String("namespace", "probler-k8s-admin", "kubernetes namespace")
	path := flag.String("path", k8sclient.DefaultAdmissionPath, "webhook HTTP path")
	flag.Parse()

	yamlBytes, err := k8sclient.ValidatingWebhookYAML(k8sclient.WebhookConfigOptions{
		Name:        *name,
		ServiceName: *serviceName,
		Namespace:   *namespace,
		Path:        *path,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	_, _ = os.Stdout.Write(yamlBytes)
}
