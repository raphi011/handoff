package handoff

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/raphi011/handoff/internal/html"
	"github.com/raphi011/handoff/internal/model"
	"golang.org/x/exp/slog"
)

type malformedRequestError struct {
	param  string
	reason string
}

func (e malformedRequestError) Error() string {
	return "malformed request param: " + e.param + " reason: " + e.reason
}

func (s *Handoff) runHTTP() error {
	router := httprouter.New()

	router.Handler("GET", "/metrics", promhttp.Handler())

	router.GET("/healthz", s.getHealth)
	router.GET("/ready", s.getReady)

	router.POST("/suites/:suite-name/runs", s.startTestSuite)
	router.POST("/suites/:suite-name/runs/:run-id", s.rerunTestSuite)
	router.GET("/suites", s.getTestSuites)
	router.GET("/suites/:suite-name/runs", s.getTestSuiteRuns)
	router.GET("/suites/:suite-name/runs/:run-id", s.getTestSuiteRun)
	router.GET("/suites/:suite-name/runs/:run-id/test/:test-name", s.getTestRunResult)

	slog.Info("Starting http server", "port", s.port)

	s.httpServer = &http.Server{
		Handler: router,
		Addr:    fmt.Sprintf(":%d", s.port),
		// TODO: set reasonable timeouts
	}

	return s.httpServer.ListenAndServe()
}

func (s *Handoff) stopHTTP() context.Context {
	httpStopCtx, cancelHttp := context.WithCancel(context.Background())

	go func() {
		timeoutCtx, cancelTimeout := context.WithTimeout(context.Background(), time.Second*30)
		defer cancelTimeout()
		defer cancelHttp()

		err := s.httpServer.Shutdown(timeoutCtx)

		if err != nil {
			slog.Warn("Http listener shutdown returned an error", "error", err)
		}
	}()

	return httpStopCtx
}

func (s *Handoff) startTestSuite(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	suite, err := s.loadTestSuite(r, p)
	if err != nil {
		s.httpError(w, err)
		return
	}

	var filter *regexp.Regexp

	if filterParam := r.URL.Query().Get("filter"); filterParam != "" {
		filter, err = regexp.Compile(filterParam)
		if err != nil {
			s.httpError(w, malformedRequestError{param: "filter", reason: "invalid regex"})
			return
		}

		if len(suite.FilterTests(filter)) == 0 {
			s.httpError(w, malformedRequestError{param: "filter", reason: "no tests match the given filter"})
		}
	}

	tsr, err := s.startNewTestSuiteRun(suite, "api", filter)
	if err != nil {
		s.httpError(w, err)
	}

	s.writeResponse(w, r, http.StatusCreated, tsr)
}

func (s *Handoff) rerunTestSuite(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	suite, err := s.loadTestSuite(r, p)
	if err != nil {
		s.httpError(w, err)
		return
	}

	var testFilter *regexp.Regexp

	if filterParam := r.URL.Query().Get("filter"); filterParam != "" {
		testFilter, err = regexp.Compile(filterParam)
		if err != nil {
			s.httpError(w, malformedRequestError{param: "filter", reason: "invalid regex"})
			return
		}
	}

	tsr, err := s.loadTestSuiteRun(context.Background(), p)
	if err != nil {
		s.httpError(w, err)
		return
	}

	s.runTestSuite(suite, tsr, testFilter, true)

	s.writeResponse(w, r, http.StatusAccepted, tsr)
}

func (s *Handoff) getTestSuites(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	testSuites := make([]model.TestSuite, len(s.readOnlyTestSuites))

	i := 0
	for _, ts := range s.readOnlyTestSuites {
		testSuites[i] = ts
		i++
	}

	s.writeResponse(w, r, http.StatusOK, testSuites)
}

func (s *Handoff) getTestSuiteRun(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	testRun, err := s.loadTestSuiteRun(r.Context(), p)
	if err != nil {
		s.httpError(w, err)
		return
	}

	s.writeResponse(w, r, http.StatusOK, testRun)
}

func (s *Handoff) getTestRunResult(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	testRun, err := s.loadTestRun(r.Context(), r, p)
	if err != nil {
		s.httpError(w, err)
		return
	}

	s.writeResponse(w, r, http.StatusOK, testRun)
}

func (s *Handoff) getHealth(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	w.WriteHeader(http.StatusOK)
}

func (s *Handoff) getReady(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	w.WriteHeader(http.StatusOK)
}

func (s *Handoff) getTestSuiteRuns(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	testRuns, err := s.loadTestSuiteRuns(r.Context(), p)
	if err != nil {
		s.httpError(w, err)
		return
	}

	if err := s.writeResponse(w, r, http.StatusOK, testRuns); err != nil {
		slog.Warn("writing get test suite runs response: %v", err)
	}
}

func (s *Handoff) writeResponse(w http.ResponseWriter, r *http.Request, status int, body any) error {
	var err error

	if headerAcceptsType(r.Header, "text/html") {
		w.Header().Add("Content-Type", "text/html")
		w.WriteHeader(status)

		switch t := body.(type) {
		case model.TestRun:
			err = html.RenderTestRun(t, w)
		case model.TestSuiteRun:
			err = html.RenderTestSuiteRun(t, w)
		case []model.TestSuiteRun:
			err = html.RenderTestSuiteRuns(t, w)
		case []model.TestSuite:
			err = html.RenderTestSuites(t, w)
		default:
			return fmt.Errorf("no template available for type %v", t)
		}
	} else {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(status)

		enc := json.NewEncoder(w)
		err = enc.Encode(body)
	}

	if err != nil {
		return fmt.Errorf("marshalling response: %w", err)
	}

	return nil
}

func headerAcceptsType(h http.Header, mimeType string) bool {
	accept := h.Get("Accept")

	return strings.Contains(accept, mimeType)
}

func (s *Handoff) loadTestSuite(r *http.Request, p httprouter.Params) (model.TestSuite, error) {
	suiteName := p.ByName("suite-name")

	ts, ok := s.readOnlyTestSuites[suiteName]

	if !ok {
		return model.TestSuite{}, model.NotFoundError{}
	}

	return ts, nil
}

func (s *Handoff) loadTestRun(ctx context.Context, r *http.Request, p httprouter.Params) (model.TestRun, error) {
	suiteName := p.ByName("suite-name")
	testName := p.ByName("test-name")
	runID, err := strconv.Atoi(p.ByName("run-id"))
	if err != nil {
		return model.TestRun{}, malformedRequestError{param: "run-id", reason: "must be an integer"}
	}

	// if len(r.URL.Query["attempt"]) != 1 {

	// }
	// attempt, err := strconv.Atoi(
	// if err != nil {
	// 	return model.TestRun{}, malformedRequestError{param: "run-id", reason: "must be an integer"}
	// }

	tr, err := s.storage.LoadTestRun(ctx, suiteName, runID, testName, 1 /* todo */)
	if err != nil {
		return model.TestRun{}, err
	}

	return tr, nil
}

func (s *Handoff) loadTestSuiteRuns(ctx context.Context, p httprouter.Params) ([]model.TestSuiteRun, error) {
	suiteName := p.ByName("suite-name")

	return s.storage.LoadTestSuiteRunsByName(ctx, suiteName)
}

func (s *Handoff) loadTestSuiteRun(ctx context.Context, p httprouter.Params) (model.TestSuiteRun, error) {
	suiteName := p.ByName("suite-name")
	runID, err := strconv.Atoi(p.ByName("run-id"))
	if err != nil {
		return model.TestSuiteRun{}, malformedRequestError{param: "run-id", reason: "must be an integer"}
	}

	tr, err := s.storage.LoadTestSuiteRun(ctx, suiteName, runID)
	if err != nil {
		return model.TestSuiteRun{}, err
	}

	tr.TestResults, err = s.storage.LoadTestRuns(ctx, suiteName, runID)
	if err != nil {
		return model.TestSuiteRun{}, err
	}

	return tr, nil
}

func (s *Handoff) httpError(w http.ResponseWriter, err error) {
	var notFound model.NotFoundError
	var malformedRequest malformedRequestError

	if errors.As(err, &notFound) {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if errors.As(err, &malformedRequest) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusInternalServerError)
	slog.Warn("internel server error", "error", err)
}
