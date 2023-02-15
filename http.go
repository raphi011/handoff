package handoff

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"sync/atomic"

	"github.com/julienschmidt/httprouter"
)

type NotFoundError struct {
}

func (e NotFoundError) Error() string {
	return "not found"
}

type MalformedRequestError struct {
	param string
}

func (e MalformedRequestError) Error() string {
	return "malformed request param: " + e.param
}

func (s *Server) runHTTP() error {
	router := httprouter.New()

	router.POST("/suite/:suite-name/run", s.StartTestSuiteRun)
	router.GET("/suite/:suite-name/run/:run-id", s.GetTestSuiteRun)

	return http.ListenAndServe("localhost:"+strconv.Itoa(s.port), router)
}

func (s *Server) httpError(w http.ResponseWriter, err error) {
	var notFound NotFoundError
	var malformedRequest MalformedRequestError

	if errors.As(err, &notFound) {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if errors.As(err, &malformedRequest) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusInternalServerError)
}

func (s *Server) StartTestSuiteRun(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	suite, err := s.getSuite(r, p)
	if err != nil {
		s.httpError(w, err)
		return
	}

	nextID := atomic.AddInt32(&s.currentRun, 1)

	event := TestRunStarted{TestRunIdentifier: TestRunIdentifier{runID: nextID, suiteName: suite.Name}}

	s.events <- event

	tr := event.Apply(TestRun{})

	body, err := json.Marshal(tr)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if _, err = w.Write(body); err != nil {
		log.Printf("error writing body: %v", err)
	}
}

func (s *Server) GetTestSuiteRun(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	tr, err := s.getTestRun(r, p)
	if err != nil {
		s.httpError(w, err)
		return
	}

	body, err := json.Marshal(tr)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if _, err = w.Write(body); err != nil {
		log.Printf("error writing body: %v", err)
	}
}

func (s *Server) getSuite(r *http.Request, p httprouter.Params) (TestSuite, error) {
	suiteName := p.ByName("suite-name")

	ts, ok := s.testSuites[suiteName]

	if !ok {
		return TestSuite{}, NotFoundError{}
	}

	return ts, nil
}

func (s *Server) getTestRun(r *http.Request, p httprouter.Params) (TestRun, error) {
	suiteName := p.ByName("suite-name")
	runID, err := strconv.Atoi(p.ByName("run-id"))
	if err != nil {
		return TestRun{}, MalformedRequestError{param: "run-id"}
	}

	tr, ok := s.testRuns.Load(testRunKey(suiteName, int32(runID)))
	if !ok {
		return TestRun{}, NotFoundError{}
	}

	return tr.(TestRun), nil
}
