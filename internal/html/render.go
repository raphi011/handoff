package html

import (
	_ "embed"
	"fmt"
	"html/template"
	"io"

	"github.com/raphi011/handoff/internal/model"
)

//go:embed test-run.tmpl
var testRunTemplate string

//go:embed test-suite-run.tmpl
var testSuiteRunTemplate string

//go:embed test-suite-runs.tmpl
var testSuiteRunsTemplate string

//go:embed test-suites.tmpl
var testSuitesTemplate string

var templatesByName map[string]*template.Template

func init() {
	templatesByName = make(map[string]*template.Template)

	templates := []struct {
		name     string
		template string
	}{
		{name: "test-run", template: testRunTemplate},
		{name: "test-suite-run", template: testSuiteRunTemplate},
		{name: "test-suite-runs", template: testSuiteRunsTemplate},
		{name: "test-suites", template: testSuitesTemplate},
	}

	for _, t := range templates {
		template, err := template.New(t.name).Parse(t.template)
		if err != nil {
			panic(fmt.Sprintf("unable to parse html template %s: %v", t.name, err))
		}

		templatesByName[t.name] = template
	}
}

func RenderTestRun(testRun model.TestRun, w io.Writer) error {
	return templatesByName["test-run"].Execute(w, testRun)
}

func RenderTestSuiteRun(testRun model.TestSuiteRun, w io.Writer) error {
	return templatesByName["test-suite-run"].Execute(w, testRun)
}

func RenderTestSuiteRuns(testRuns []model.TestSuiteRun, w io.Writer) error {
	return templatesByName["test-suite-runs"].Execute(w, testRuns)
}

func RenderTestSuites(testSuites []model.TestSuite, w io.Writer) error {
	return templatesByName["test-suites"].Execute(w, testSuites)
}
