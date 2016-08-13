package acceptance_test

import (
	"bytes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/garden"
	garden_client "code.cloudfoundry.org/garden/client"
	garden_connection "code.cloudfoundry.org/garden/client/connection"
)

var _ = Describe("Flannel with BOSH links", func() {
	var hosts []string = []string{"10.244.99.10", "10.244.99.11"}

	getClient := func(host string) garden.Client {
		return garden_client.New(garden_connection.New("tcp", host+":7777"))
	}

	createContainer := func(client garden.Client) garden.Container {
		container, err := client.Create(garden.ContainerSpec{})
		Expect(err).NotTo(HaveOccurred())
		return container
	}

	It("allows connections between containers on different hosts", func() {
		client0, client1 := getClient(hosts[0]), getClient(hosts[1])

		By("creating two containers")
		container0, container1 := createContainer(client0), createContainer(client1)

		info1, err := container1.Info()
		Expect(err).NotTo(HaveOccurred())

		stdoutBuffer := &bytes.Buffer{}

		By("pinging from one to the other")
		pingProcess, err := container0.Run(garden.ProcessSpec{
			Path: "/bin/ping",
			Args: []string{"-c1", "-w1", info1.ContainerIP},
		}, garden.ProcessIO{
			Stdout: stdoutBuffer,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(pingProcess.Wait()).To(Equal(0))

		By("destroying both containers")
		Expect(client0.Destroy(container0.Handle())).To(Succeed())
		Expect(client1.Destroy(container1.Handle())).To(Succeed())
	})
})
