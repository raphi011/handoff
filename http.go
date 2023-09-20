package handoff

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/raphi011/handoff/internal/html"
	"github.com/raphi011/handoff/internal/model"
	"golang.org/x/exp/slog"
)

type notFoundError struct{}

func (e notFoundError) Error() string {
	return "not found"
}

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

	router.POST("/suites/:suite-name/runs", s.startTestSuiteRun)
	router.GET("/suites", s.getTestSuites)
	router.GET("/suites/:suite-name/runs", s.getTestSuiteRuns)
	router.GET("/suites/:suite-name/runs/:run-id", s.getTestSuiteRun)

	slog.Info("Starting http server", "port", s.port)

	return http.ListenAndServe("localhost:"+strconv.Itoa(s.port), router)
}

func (s *Handoff) startTestSuiteRun(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	suite, err := s.getSuite(r, p)
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
	}

	event := testRunStartedEvent{
		testRunIdentifier: testRunIdentifier{runID: s.nextID(), suiteName: suite.Name},
		scheduled:         time.Now(),
		triggeredBy:       "http",
		testFilter:        filter,
		tests:             len(suite.Tests),
	}

	s.events <- event

	tr := event.Apply(model.TestSuiteRun{})

	// TODO: set status created code
	s.writeResponse(w, r, tr)
}

func (s *Handoff) getTestSuites(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	testSuites := make([]model.TestSuite, len(s.testSuites))

	i := 0
	for _, ts := range s.testSuites {
		testSuites[i] = ts
		i++
	}

	s.writeResponse(w, r, testSuites)
}

func (s *Handoff) getTestSuiteRun(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	testRun, err := s.getTestRun(p)
	if err != nil {
		s.httpError(w, err)
		return
	}

	s.writeResponse(w, r, testRun)
}

func (s *Handoff) getHealth(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	w.WriteHeader(http.StatusOK)
}

func (s *Handoff) getReady(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	w.WriteHeader(http.StatusOK)
}

func (s *Handoff) getTestSuiteRuns(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	testRuns := s.getTestRuns(p)

	if err := s.writeResponse(w, r, testRuns); err != nil {
		slog.Warn("writing get test suite runs response: %v", err)
	}
}

func (s *Handoff) writeResponse(w http.ResponseWriter, r *http.Request, body any) error {
	var err error

	if headerAcceptsType(r.Header, "text/html") {
		w.Header().Add("Content-Type", "text/html")

		switch t := body.(type) {
		case model.TestSuiteRun:
			html.RenderTestRun(t, w)
		case []model.TestSuiteRun:
			html.RenderTestRuns(t, w)
		case []model.TestSuite:
			html.RenderTestSuites(t, w)
		default:
			return fmt.Errorf("no template available for type %v", t)
		}
	} else {
		w.Header().Add("Content-Type", "application/json")

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

func (s *Handoff) getSuite(r *http.Request, p httprouter.Params) (model.TestSuite, error) {
	suiteName := p.ByName("suite-name")

	ts, ok := s.testSuites[suiteName]

	if !ok {
		return model.TestSuite{}, notFoundError{}
	}

	return ts, nil
}

func (s *Handoff) nextID() int32 {
	return atomic.AddInt32(&s.currentRun, 1)
}

func (s *Handoff) getTestRun(p httprouter.Params) (model.TestSuiteRun, error) {
	suiteName := p.ByName("suite-name")
	runID, err := strconv.Atoi(p.ByName("run-id"))
	if err != nil {
		return model.TestSuiteRun{}, malformedRequestError{param: "run-id", reason: "must be an integer"}
	}

	tr, ok := s.testSuiteRuns.Load(testRunKey(suiteName, int32(runID)))
	if !ok {
		return model.TestSuiteRun{}, notFoundError{}
	}

	return tr.(model.TestSuiteRun), nil
}

func (s *Handoff) getTestRuns(p httprouter.Params) []model.TestSuiteRun {
	suiteName := p.ByName("suite-name")

	testRuns := []model.TestSuiteRun{}

	s.testSuiteRuns.Range(func(key, value any) bool {
		if k := key.(string); strings.HasPrefix(k, suiteName+"-") {
			testRuns = append(testRuns, value.(model.TestSuiteRun))
		}

		return true
	})

	return testRuns
}

func (s *Handoff) httpError(w http.ResponseWriter, err error) {
	var notFound notFoundError
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
