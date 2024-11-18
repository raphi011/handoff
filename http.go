package handoff

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/raphi011/handoff/internal/html"
	"github.com/raphi011/handoff/internal/model"
)

type malformedRequestError struct {
	param  string
	reason string
}

func (e malformedRequestError) Error() string {
	return "malformed request param: " + e.param + " reason: " + e.reason
}

func (s *Server) runHTTP() error {
	router := httprouter.New()

	router.Handler("GET", "/metrics", promhttp.Handler())

	if s.config.EnablePprof {
		router.Handler(http.MethodGet, "/debug/pprof/*item", http.DefaultServeMux)
	}

	router.GET("/healthz", s.getHealth)
	router.GET("/ready", s.getReady)

	router.POST("/suites/:suite-name/runs", s.startTestSuite)
	router.GET("/suites", s.getTestSuites)
	router.GET("/suites/:suite-name/runs", s.getTestSuiteRuns)
	router.GET("/suites/:suite-name/runs/:run-id", s.getTestSuiteRun)
	router.GET("/suites/:suite-name/runs/:run-id/test/:test-name", s.getTestRunResult)

	router.GET("/schedules", s.getSchedules)
	router.POST("/schedules/:schedule-name", s.createSchedule)
	router.DELETE("/schedules/:schedule-name", s.deleteSchedule)

	s.httpServer = &http.Server{
		Handler: router,
		// TODO: set reasonable timeouts

		// needed for debug/pprof/profile endpoint
		WriteTimeout: 31 * time.Second,
	}

	l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.config.HostIP, s.config.Port))
	if err != nil {
		return fmt.Errorf("listening on port %d: %w", s.config.Port, err)
	}

	if s.config.Port == 0 {
		// if we are using a randomly assigned port put it back into the config
		// so that e.g. tests know where to send requests to.
		s.config.Port = l.Addr().(*net.TCPAddr).Port
	}

	s.log.Info("Starting http server", "host", s.config.HostIP, "port", s.config.Port)

	go func() {
		err = s.httpServer.Serve(l)
		if err != nil && err != http.ErrServerClosed {
			s.log.Error("http server failed", "error", err)
		}
	}()

	return nil
}

func (s *Server) stopHTTP() chan error {
	httpStopped := make(chan error)

	go func() {
		timeoutCtx, cancelTimeout := context.WithTimeout(context.Background(), time.Second*30)
		defer cancelTimeout()

		err := s.httpServer.Shutdown(timeoutCtx)

		httpStopped <- err
	}()

	return httpStopped
}

func (s *Server) startTestSuite(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	ts, err := s.loadTestSuite(r, p)
	if err != nil {
		s.httpError(w, err)
		return
	}

	reference := r.URL.Query().Get("ref")

	filter, err := filterParam(ts, r)
	if err != nil {
		s.httpError(w, err)
		return
	}

	tsr, err := s.startNewTestSuiteRun(ts, model.RunParams{
		TriggeredBy: "api",
		TestFilter:  filter,
		Reference:   reference,
	})
	if err != nil {
		s.httpError(w, err)
	}

	s.writeResponse(w, r, http.StatusCreated, tsr)
}

func (s *Server) getSchedules(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	var schedules []model.ScheduledRun

	schedules = append(schedules, s.readOnlySchedules...)

	s.writeResponse(w, r, http.StatusOK, schedules)
}

func (s *Server) deleteSchedule(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	scheduleName := p.ByName("schedule-name")
	if scheduleName == "" {
		s.httpError(w, malformedRequestError{":schedule-name", "must provide a schedule name to delete"})
		return
	}

	err := s.storage.DeleteScheduledRun(context.Background(), scheduleName)
	if err != nil {
		s.httpError(w, fmt.Errorf("failed to delete scheduled run: %w", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) createSchedule(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	scheduleName := p.ByName("schedule-name")

	ts, err := s.loadTestSuite(r, p)
	if err != nil {
		s.httpError(w, err)
		return
	}
	schedule := r.Header.Get("schedule")
	filter, err := filterParam(ts, r)
	if err != nil {
		s.httpError(w, err)
		return
	}

	sr := model.ScheduledRun{
		Name:          scheduleName,
		TestSuiteName: ts.Name,
		Schedule:      schedule,
		TestFilter:    filter,
	}

	if _, err := s.startSchedule(sr, true); err != nil {
		s.httpError(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func filterParam(ts model.TestSuite, r *http.Request) (*regexp.Regexp, error) {
	filter := r.URL.Query().Get("filter")
	if filter == "" {
		return nil, nil
	}

	filterRegex, err := regexp.Compile(filter)
	if err != nil {
		return nil, malformedRequestError{param: "filter", reason: "invalid regex"}
	}

	if len(ts.FilterTests(filterRegex)) == 0 {
		return nil, malformedRequestError{param: "filter", reason: "no tests match the given filter"}
	}

	return filterRegex, nil
}

func (s *Server) getTestSuites(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	testSuites := make([]model.TestSuite, len(s.readOnlyTestSuites))

	i := 0
	for _, ts := range s.readOnlyTestSuites {
		testSuites[i] = ts
		i++
	}

	s.writeResponse(w, r, http.StatusOK, testSuites)
}

func (s *Server) getTestSuiteRun(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	testRun, err := s.loadTestSuiteRun(r.Context(), p)
	if err != nil {
		s.httpError(w, err)
		return
	}

	s.writeResponse(w, r, http.StatusOK, testRun)
}

func (s *Server) getTestRunResult(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	testRun, err := s.loadTestRuns(r.Context(), r, p)
	if err != nil {
		s.httpError(w, err)
		return
	}

	s.writeResponse(w, r, http.StatusOK, testRun)
}

func (s *Server) getHealth(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	w.WriteHeader(http.StatusOK)
}

func (s *Server) getReady(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	w.WriteHeader(http.StatusOK)
}

func (s *Server) getTestSuiteRuns(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	testRuns, err := s.loadTestSuiteRuns(r.Context(), p)
	if err != nil {
		s.httpError(w, err)
		return
	}

	if err := s.writeResponse(w, r, http.StatusOK, testRuns); err != nil {
		s.log.Warn("writing get test suite runs response", "error", err)
	}
}

func (s *Server) writeResponse(w http.ResponseWriter, r *http.Request, status int, body any) error {
	var err error

	if headerAcceptsType(r.Header, "text/html") {
		w.Header().Add("Content-Type", "text/html")
		w.WriteHeader(status)

		switch t := body.(type) {
		case model.TestRun:
			err = html.RenderTestRun(t).Render(r.Context(), w)
		case []model.ScheduledRun:
			err = html.RenderSchedules(t).Render(r.Context(), w)
		case model.TestSuiteRun:
			err = html.RenderTestSuiteRun(t).Render(r.Context(), w)
		case []model.TestSuiteRun:
			err = html.RenderTestSuiteRuns(t).Render(r.Context(), w)
		case []model.TestSuite:
			err = html.RenderTestSuites(t).Render(r.Context(), w)
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

func (s *Server) loadTestSuite(r *http.Request, p httprouter.Params) (model.TestSuite, error) {
	suiteName := p.ByName("suite-name")

	ts, ok := s.readOnlyTestSuites[suiteName]

	if !ok {
		return model.TestSuite{}, model.NotFoundError{}
	}

	return ts, nil
}

func (s *Server) loadTestRuns(ctx context.Context, r *http.Request, p httprouter.Params) ([]model.TestRun, error) {
	suiteName := p.ByName("suite-name")
	testName := p.ByName("test-name")
	runID, err := strconv.Atoi(p.ByName("run-id"))
	if err != nil {
		return []model.TestRun{}, malformedRequestError{param: "run-id", reason: "must be an integer"}
	}

	tsr, err := s.storage.LoadTestSuiteRun(ctx, suiteName, runID)
	if err != nil {
		return []model.TestRun{}, err
	}

	tr := tsr.TestRunsByName(testName)

	return tr, nil
}

func (s *Server) loadTestSuiteRuns(ctx context.Context, p httprouter.Params) ([]model.TestSuiteRun, error) {
	suiteName := p.ByName("suite-name")

	return s.storage.LoadTestSuiteRunsByName(ctx, suiteName)
}

func (s *Server) loadTestSuiteRun(ctx context.Context, p httprouter.Params) (model.TestSuiteRun, error) {
	suiteName := p.ByName("suite-name")
	runID, err := strconv.Atoi(p.ByName("run-id"))
	if err != nil {
		return model.TestSuiteRun{}, malformedRequestError{param: "run-id", reason: "must be an integer"}
	}

	tr, err := s.storage.LoadTestSuiteRun(ctx, suiteName, runID)
	if err != nil {
		return model.TestSuiteRun{}, err
	}

	return tr, nil
}

func (s *Server) httpError(w http.ResponseWriter, err error) {
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
	s.log.Warn("internal server error", "error", err)
}
