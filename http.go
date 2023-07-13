package handoff

import (
	_ "embed"
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

//go:embed testrun.tmpl
var testRunTemplate string
//go:embed testruns.tmpl
var testRunsTemplate string

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

	router.Handler("GET", "/metrics", promhttp.Handler())

	router.POST("/suite/:suite-name/run", s.StartTestSuiteRun)
	router.GET("/suite/:suite-name/run/:run-id", s.GetTestSuiteRun)

	router.GET("/runs", s.GetResults)
   router.GET("/runs/:run-id", s.GetResult)

	log.Printf("Running server at port %d\n", s.port)

	return http.ListenAndServe("localhost:"+strconv.Itoa(s.port), router)
}

func (s *Server) StartTestSuiteRun(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	suite, err := s.getSuite(r, p)
	if err != nil {
		s.httpError(w, err)
		return
	}

	var filter *regexp.Regexp

	if filterParam := r.URL.Query().Get("filter"); filterParam != "" {
		filter, err = regexp.Compile(filterParam)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	event := TestRunStartedEvent{
		TestRunIdentifier: TestRunIdentifier{runID: s.nextID(), suiteName: suite.Name},
		Scheduled:         time.Now(),
		TriggeredBy:       "http",
		TestFilter:        filter,
		Tests:             len(suite.Tests),
	}

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

	w.WriteHeader(http.StatusCreated)
}

func (s *Server) GetResults(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	var testRun TestRun
	s.testRuns.Range(func(key, value any) bool {
		testRun = value.(TestRun)
		return false
	})

	if testRun.SuiteName == "" {
		w.Write([]byte("No testruns found"))
		return
	}

	tmpl, err := template.New("results").Parse(testRunTemplate)
	if err != nil {
		s.httpError(w, err)
		return
	}

   // ignore error for now
	err = tmpl.Execute(w, testRun)
   if err != nil {
      log.Printf("error executing template %v", err)
   }
}

func (s *Server) GetResult(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
   runID := p.ByName("run-id")

   testRun, ok := s.testRuns.Load(runID)
   if !ok {
     s.httpError(w, NotFoundError{})
     return
   }


	tmpl, err := template.New("results").Parse(testRunTemplate)
   
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

func (s *Server) nextID() int32 {
	return atomic.AddInt32(&s.currentRun, 1)
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
