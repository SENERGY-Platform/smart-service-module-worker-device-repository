package tests

import (
	"context"
	"github.com/SENERGY-Platform/smart-service-module-worker-device-repository/pkg"
	"github.com/SENERGY-Platform/smart-service-module-worker-device-repository/pkg/worker"
	"github.com/SENERGY-Platform/smart-service-module-worker-device-repository/tests/mocks"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/configuration"
	"os"
	"sync"
	"testing"
	"time"
)

const TEST_CASE_DIR = "./testcases/"

func TestWithMocks(t *testing.T) {
	libConf, err := configuration.LoadLibConfig("../config.json")
	if err != nil {
		t.Error(err)
		return
	}
	conf, err := configuration.Load[worker.Config]("../config.json")
	if err != nil {
		t.Error(err)
		return
	}
	libConf.CamundaWorkerWaitDurationInMs = 200

	infos, err := os.ReadDir(TEST_CASE_DIR)
	if err != nil {
		t.Error(err)
		return
	}
	for _, info := range infos {
		name := info.Name()
		if info.IsDir() && isValidaForMockTest(TEST_CASE_DIR+name) {
			t.Run(name, func(t *testing.T) {
				runTest(t, TEST_CASE_DIR+name, conf, libConf)
			})
		}
	}
}

func isValidaForMockTest(dir string) bool {
	expectedFiles := []string{
		"camunda_tasks.json",
		"device_repository_responses.json",
		"device_repository_requests.json",
		"device_selection_responses.json",
		"device_selection_requests.json",
		"expected_smart_service_repo_requests.json",
	}
	infos, err := os.ReadDir(dir)
	if err != nil {
		panic(err)
	}
	files := map[string]bool{}
	for _, info := range infos {
		if !info.IsDir() {
			files[info.Name()] = true
		}
	}
	for _, expected := range expectedFiles {
		if !files[expected] {
			return false
		}
	}
	return true
}

func runTest(t *testing.T, testCaseLocation string, config worker.Config, libConf configuration.Config) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	camunda := mocks.NewCamundaMock()
	libConf.CamundaUrl = camunda.Start(ctx, wg)
	err := camunda.AddFileToQueue(testCaseLocation + "/camunda_tasks.json")
	if err != nil {
		t.Error(err)
		return
	}

	libConf.AuthEndpoint = mocks.Keycloak(ctx, wg)

	deviceRepo := mocks.HttpService{}
	err = deviceRepo.SetResponsesFromFile(testCaseLocation + "/device_repository_responses.json")
	if err != nil {
		t.Error(err)
		return
	}
	config.DeviceManagerUrl = deviceRepo.Start(ctx, wg)

	deviceSelection := mocks.HttpService{}
	err = deviceSelection.SetResponsesFromFile(testCaseLocation + "/device_selection_responses.json")
	if err != nil {
		t.Error(err)
		return
	}
	config.DeviceSelectionUrl = deviceSelection.Start(ctx, wg)

	smartServiceRepo := mocks.NewSmartServiceRepoMock(libConf, config)
	libConf.SmartServiceRepositoryUrl = smartServiceRepo.Start(ctx, wg)

	err = pkg.Start(ctx, wg, config, libConf)
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(1 * time.Second)

	err = deviceRepo.CheckExpectedRequestsFromFileLocation(testCaseLocation + "/device_repository_requests.json")
	if err != nil {
		t.Error("/device_repository_requests.json", err)
	}

	err = deviceSelection.CheckExpectedRequestsFromFileLocation(testCaseLocation + "/device_selection_requests.json")
	if err != nil {
		t.Error("/device_selection_requests.json", err)
	}

	err = smartServiceRepo.CheckExpectedRequestsFromFileLocation(testCaseLocation + "/expected_smart_service_repo_requests.json")
	if err != nil {
		t.Error("/expected_smart_service_repo_requests.json", err)
	}
}
