package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"lib/models"
	"lib/policy_client"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"code.cloudfoundry.org/lager"
)

func main() {
	// Our GA scalability target is: 100 cells, 100 apps and 200 instances per app with 3 policies per app.

	// config:
	// - total policies (default: 60,000. 10000 apps and 2 instances per app with 3 policies per app)

	// - number of cells (default: 100)
	// - policies per cell (default: 600 src, 600 dst) // policies assumed to be uniform/bi-directional
	// - containers per cell (default: 200) // 200 unique app ids per cell
	// - polling frequency (default 5)
	// - run forever unitl Ctrl+C? or set some duration?

	// can 1 workstation generate enough load?
	// 1 request per cell, every 5 seconds (* 100 cells)
	// 100 requests, every 5 seconds
	// 20 requests per second

	// before test run:
	// clean up policies
	// if necessary, disable cleanup (restart server with long cleanup polling interval)
	//
	// setup
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	logger := lager.NewLogger("container-networking.policy-server-test")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.INFO))
	logger.Info("started")
	defer logger.Info("exited")

	// Parse flags
	var (
		apps, numCells, policiesPerApp int
		pollInterval                   time.Duration
		token, policyServerAPI         string
		setup                          bool
	)
	flag.IntVar(&apps, "apps", 10000, "number of apps")
	flag.IntVar(&numCells, "numCells", 100, "number of cells")
	// TODO app instances
	flag.IntVar(&policiesPerApp, "policiesPerApp", 3, "policies per app")
	flag.DurationVar(&pollInterval, "pollInterval", 5*time.Second, "polling interval on each cell")
	flag.StringVar(&token, "token", "", "OAuth for policy server")
	flag.StringVar(&policyServerAPI, "api", "", "policy server base URL")
	flag.BoolVar(&setup, "setup", true, "if true, remove existing policies and create new policies")
	flag.Parse()

	if policyServerAPI == "" {
		logger.Fatal("Specify policy server", errors.New(""))
	}

	// Initialize policy client
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // insecure!!!
			},
		},
	}
	client := policy_client.NewExternal(logger, httpClient, policyServerAPI)

	// Purge existing policies
	if setup {
		logger.Info("getting-existing-policies")
		policies, err := client.GetPolicies(token)
		logger.Info("existing-policies", lager.Data{"num-existing-policies": len(policies)})
		if err != nil {
			logger.Fatal("Failed to get policies", err)
		}
		logger.Info("deleting-existing-policies")
		err = client.DeletePolicies(token, policies)
		if err != nil {
			logger.Fatal("Failed to delete policies", err)
		}
		logger.Info("done-deleting-existing-policies")
	} else {
		logger.Info("not-cleaning-up-existing-policies")
	}

	// creates "applications" (10,000 guids) (in local memory)
	logger.Info("creating-applications")
	var guids []string
	for i := 0; i < apps; i++ {
		guid := fmt.Sprintf("9cb281b-e272-4df7-b398-b6663ca-%04d", i) // TODO we should do better... indexes and what not
		guids = append(guids, guid)
	}
	logger.Info("done-creating-applications")

	if setup {
		// creates the policies (30,000) (using the fake app guids)
		policies := []models.Policy{}
		logger.Info("creating-policies")
		for _, guid := range guids {
			for i := 0; i < policiesPerApp; i++ {
				policy := models.Policy{
					Source: models.Source{
						ID: guid,
					},
					Destination: models.Destination{
						ID:       guid, // TODO make this random or explore other distrubutions (eg hotspot)
						Protocol: "tcp",
						Port:     9000 + i,
					},
				}
				policies = append(policies, policy)
			}
		}
		err := client.AddPolicies(token, policies)
		if err != nil {
			logger.Fatal("Failed to create policies", err)
		}
		logger.Info("done-creating-policies")
	} else {
		logger.Info("skipping-creating-policies")
	}

	// simulate placing "app instances on cells" (100 cells, with 100 app guids per cell)
	appsPerCell := apps / numCells
	var cells [][]string
	for i := 0; i < numCells; i++ {
		cells = append(cells, guids[i*appsPerCell:(i+1)*appsPerCell])
	}

	// each "cell" is its own goroutine which spawns goroutines to make requests
	callPolicyServer := func(ids []string, index, numCalls int) {
		_, err := client.GetPoliciesByID(token, ids...)
		if err != nil {
			logger.Error("failed-to-get-policies", err)
		} else {
			logger.Info(fmt.Sprintf("completed-request-from-cell-%d-call-%d", index, numCalls))
		}
	}

	pollPolicyServer := func(ids []string, index int) {
		numCalls := 0
		for {
			select {
			case <-time.After(pollInterval): // TODO jitter?
				go callPolicyServer(ids, index, numCalls)
				numCalls = numCalls + 1
				continue
			}
		}
	}

	// each "cell" makes requests to server for it's app instances
	for i := 0; i < len(cells); i++ {
		go func(i int) {
			logger.Info(fmt.Sprintf("starting-from-cell-%d", i))
			pollPolicyServer(cells[i], i)
		}(i)
	}

	fmt.Println("Press CTRL-C to exit")
	for {
		time.Sleep(10 * time.Second)
	}
}
