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
	url := c.url("/suites/%s/runs", suiteName)
	if filter != nil {
		url += fmt.Sprintf("?filter=%s", filter)
	}

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return TestSuiteRun{}, err
	}

	var tsr TestSuiteRun

	err = c.do(ctx, req, &tsr)
	if err != nil {
		return TestSuiteRun{}, err
	}

	return tsr, nil
}

func (c Client) GetTestSuiteRun(ctx context.Context, suiteName string, runID int) (TestSuiteRun, error) {
	url := c.url("/suites/%s/runs/%d", suiteName, runID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return TestSuiteRun{}, err
	}

	var tsr TestSuiteRun

	if err = c.do(ctx, req, &tsr); err != nil {
		return TestSuiteRun{}, err
	}

	return tsr, nil
}

func (c Client) GetTestRun(ctx context.Context, suiteName string, runID int, testName string) ([]TestRun, error) {
	url := c.url("/suites/%s/runs/%d/test/%s", suiteName, runID, testName)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return []TestRun{}, err
	}

	var tr []TestRun

	if err = c.do(ctx, req, &tr); err != nil {
		return []TestRun{}, err
	}

	return tr, nil
}

func (c Client) url(path string, args ...any) string {
	return fmt.Sprintf(c.host+path, args...)
}

func (c Client) do(ctx context.Context, req *http.Request, body any) error {
	req = req.WithContext(ctx)
	req.Header.Add("Accept", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return RequestError{res.StatusCode}
	}

	if body != nil {
		d := json.NewDecoder(res.Body)

		if err = d.Decode(body); err != nil {
			return err
		}
	}

	return nil
}
