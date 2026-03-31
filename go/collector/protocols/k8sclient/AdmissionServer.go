package k8sclient

import (
	"os"
	"strconv"

	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8web/go/web/server"
)

func (c *ClientGoCollector) StartAdmissionServer(resources ifs.IResources) error {
	port, err := envInt("ADMISSION_PORT", 8443)
	if err != nil {
		return err
	}

	serverConfig := &server.RestServerConfig{
		Host:           envString("ADMISSION_HOST", "0.0.0.0"),
		Port:           port,
		Authentication: false,
		Prefix:         "",
		CertName:       envString("ADMISSION_CERT_NAME", "/data/admission"),
	}
	svr, err := server.NewRestServer(serverConfig)
	if err != nil {
		return err
	}
	err = c.RegisterAdmissionHandler(svr.(*server.RestServer), envString("ADMISSION_PATH", DefaultAdmissionPath))
	if err != nil {
		return err
	}
	go func() {
		if startErr := svr.Start(); startErr != nil {
			resources.Logger().Error("Admission web server stopped: ", startErr.Error())
		}
	}()
	return nil
}

func envString(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func envInt(key string, fallback int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	return strconv.Atoi(value)
}
