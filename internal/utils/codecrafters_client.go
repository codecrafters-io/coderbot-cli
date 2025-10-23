package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	retry "github.com/avast/retry-go"
	"github.com/getsentry/sentry-go"
	"github.com/levigross/grequests"
)

type CreateTestRunResponse struct {
	ID string `json:"id"`

	// BuildLogstreamURL is returned when the test run is waiting on a build
	PendingBuildID           string `json:"pending_build_id"`
	PendingBuildLogstreamURL string `json:"pending_build_logstream_url"`

	// LogstreamURL contains test logs.
	LogstreamURL string `json:"logstream_url"`

	ErrorMessage string `json:"error_message"`
	IsError      bool   `json:"is_error"`
}

type FetchBuildStatusResponse struct {
	Status string `json:"status"`

	ErrorMessage string `json:"error_message"`
	IsError      bool   `json:"is_error"`
}

type FetchTestRunResponse struct {
	Status string `json:"status"`

	ErrorMessage string `json:"error_message"`
	IsError      bool   `json:"is_error"`
}

type CodecraftersClient struct {
	ServerUrl string
}

func NewCodecraftersClient(serverUrl string) CodecraftersClient {
	return CodecraftersClient{ServerUrl: serverUrl}
}

func (c CodecraftersClient) headers() map[string]string {
	return map[string]string{
		"X-Codecrafters-CLI-Version": VersionString(),
	}
}

func (c CodecraftersClient) CreateTestRun(autofixRequestId string, commitSha string) (CreateTestRunResponse, error) {
	response, err := grequests.Post(c.ServerUrl+"/services/coderbot/create_test_run", &grequests.RequestOptions{
		JSON: map[string]interface{}{
			"autofix_request_id": autofixRequestId,
			"commit_sha":         commitSha,
		},
		Headers: c.headers(),
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create test run: %s\n", err)
		return CreateTestRunResponse{}, err
	}

	if !response.Ok && response.StatusCode != 403 {
		return CreateTestRunResponse{}, fmt.Errorf("failed to create test run. status code: %d, body: %s", response.StatusCode, response.String())
	}

	createTestRunResponse := CreateTestRunResponse{}

	err = json.Unmarshal(response.Bytes(), &createTestRunResponse)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create test run: %s\n", err)
		return CreateTestRunResponse{}, err
	}

	if createTestRunResponse.IsError {
		return createTestRunResponse, fmt.Errorf("%s", createTestRunResponse.ErrorMessage)
	}

	return createTestRunResponse, nil
}

func (c CodecraftersClient) FetchTestRun(testRunID string) (FetchTestRunResponse, error) {
	var fetchTestRunResponse FetchTestRunResponse

	err := retry.Do(
		func() error {
			var err error
			fetchTestRunResponse, err = c.doFetchTestRun(testRunID)
			if err != nil {
				return err
			}

			if fetchTestRunResponse.Status != "failure" && fetchTestRunResponse.Status != "success" {
				return fmt.Errorf("unexpected test run status: %s", fetchTestRunResponse.Status)
			}

			return nil
		},
		retry.Attempts(5),
		retry.DelayType(retry.BackOffDelay),
		retry.MaxDelay(2*time.Second),
		retry.Delay(500*time.Millisecond),
		retry.LastErrorOnly(true),
	)

	if err != nil {
		return FetchTestRunResponse{}, err
	}

	return fetchTestRunResponse, nil
}

func (c CodecraftersClient) doFetchTestRun(testRunID string) (FetchTestRunResponse, error) {
	response, err := grequests.Get(fmt.Sprintf("%s/services/coderbot/fetch_test_run", c.ServerUrl), &grequests.RequestOptions{
		Params: map[string]string{
			"test_run_id": testRunID,
		},
		Headers: c.headers(),
	})

	if err != nil {
		return FetchTestRunResponse{}, fmt.Errorf("failed to fetch test run result from CodeCrafters: %s", err)
	}

	if !response.Ok {
		return FetchTestRunResponse{}, fmt.Errorf("failed to fetch test run result from CodeCrafters. status code: %d", response.StatusCode)
	}

	fetchTestRunResponse := FetchTestRunResponse{}

	err = json.Unmarshal(response.Bytes(), &fetchTestRunResponse)
	if err != nil {
		return FetchTestRunResponse{}, fmt.Errorf("failed to fetch test run result from CodeCrafters: %s", err)
	}

	return fetchTestRunResponse, nil
}

func (c CodecraftersClient) FetchBuild(buildId string) (FetchBuildStatusResponse, error) {
	var fetchBuildResponse FetchBuildStatusResponse

	err := retry.Do(
		func() error {
			var err error
			fetchBuildResponse, err = c.doFetchBuild(buildId)
			if err != nil {
				return err
			}

			if fetchBuildResponse.Status != "failure" && fetchBuildResponse.Status != "success" {
				return fmt.Errorf("unexpected build status: %s", fetchBuildResponse.Status)
			}

			return nil
		},
		retry.Attempts(11),
		retry.DelayType(retry.BackOffDelay),
		retry.MaxDelay(2*time.Second),
		retry.Delay(100*time.Millisecond),
		retry.LastErrorOnly(true),
	)

	if err != nil {
		if fetchBuildResponse.Status != "failure" && fetchBuildResponse.Status != "success" {
			sentry.CaptureException(err)
		}

		return FetchBuildStatusResponse{}, err
	}

	return fetchBuildResponse, nil
}

func (c CodecraftersClient) doFetchBuild(buildId string) (FetchBuildStatusResponse, error) {
	response, err := grequests.Get(fmt.Sprintf("%s/services/coderbot/fetch_test_runner_build", c.ServerUrl), &grequests.RequestOptions{
		Params: map[string]string{
			"test_runner_build_id": buildId,
		},
		Headers: c.headers(),
	})

	if err != nil {
		return FetchBuildStatusResponse{}, fmt.Errorf("failed to fetch build result from CodeCrafters: %s", err)
	}

	if !response.Ok {
		return FetchBuildStatusResponse{}, fmt.Errorf("failed to fetch build result from CodeCrafters. status code: %d", response.StatusCode)
	}

	fetchBuildResponse := FetchBuildStatusResponse{}

	err = json.Unmarshal(response.Bytes(), &fetchBuildResponse)
	if err != nil {
		return FetchBuildStatusResponse{}, fmt.Errorf("failed to fetch build result from CodeCrafters: %s", err)
	}

	return fetchBuildResponse, nil
}
