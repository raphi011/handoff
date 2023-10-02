package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	"github.com/raphi011/handoff/internal/model"
)

type TestSuiteRun = model.TestSuiteRunHTTP
type TestRun = model.TestRunHTTP

type Client struct {
	http *http.Client
	host string
}

type RequestError struct {
	ResponseCode int
}

func (e RequestError) Error() string {
	return fmt.Sprintf("request failed with status %d", e.ResponseCode)
}

func New(host string, c *http.Client) Client {
	return Client{http: c, host: host}
}

func (c Client) CreateTestSuiteRun(ctx context.Context, suiteName string, filter *regexp.Regexp) (TestSuiteRun, error) {
	res, err := c.http.Post(c.host+fmt.Sprintf("/suites/%s/runs", suiteName), "", nil)
	if err != nil {
		return TestSuiteRun{}, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		return TestSuiteRun{}, RequestError{res.StatusCode}
	}

	var tsr TestSuiteRun

	d := json.NewDecoder(res.Body)

	if err = d.Decode(&tsr); err != nil {
		return TestSuiteRun{}, err
	}

	return tsr, nil
}

func (c Client) GetTestSuiteRun(ctx context.Context, suiteName string, runID int) (TestSuiteRun, error) {
	url := c.host + fmt.Sprintf("/suites/%s/runs/%d", suiteName, runID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return TestSuiteRun{}, err
	}

	req = req.WithContext(ctx)

	res, err := c.http.Do(req)
	if err != nil {
		return TestSuiteRun{}, err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 && res.StatusCode >= 300 {
		return TestSuiteRun{}, RequestError{res.StatusCode}
	}

	var tsr TestSuiteRun

	d := json.NewDecoder(res.Body)

	if err = d.Decode(&tsr); err != nil {
		return TestSuiteRun{}, err
	}

	return tsr, nil
}
