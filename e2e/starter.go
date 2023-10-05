package e2e

import (
	"context"
	"fmt"
	"os"
	"path"

	tc "github.com/testcontainers/testcontainers-go/modules/compose"

	"github.com/g41797/sputnik/sidecar"
)

func StartServices() (stopServices func(), err error) {

	confFolder, err := sidecar.ConfFolder()
	if err != nil {
		return nil, err
	}

	composePath := path.Join(confFolder, "docker-compose.yml")

	_, err = os.ReadFile(composePath)

	if err != nil {
		fmt.Println("docker-compose.yml does not exist. Please start required services manually")
		stopServices = func() {
			return
		}
		return stopServices, nil
	}

	dc, err := tc.NewDockerCompose(composePath)
	if err != nil {
		return nil, err
	}

	dc.Up(context.Background(), tc.Wait(true))

	services := dc.Services()
	fmt.Println(services)

	stopServices = func() {
		dc.Down(context.Background())
	}

	return stopServices, nil
}
