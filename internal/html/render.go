package html

import (
	_ "embed"
	"fmt"
	"html/template"
	"io"

	"github.com/raphi011/handoff/internal/model"
)

//go:embed testrun.tmpl
var testRunTemplate string

//go:embed testruns.tmpl
var testRunsTemplate string

var templatesByName map[string]*template.Template

func init() {
	templatesByName = make(map[string]*template.Template)

	templates := []struct {
		name     string
		template string
	}{
		{name: "test-run", template: testRunTemplate},
		{name: "test-runs", template: testRunsTemplate},
	}

	for _, t := range templates {
		template, err := template.New(t.name).Parse(t.template)
		if err != nil {
			panic(fmt.Sprintf("unable to parse html template %s: %v", t.name, err))
		}

		templatesByName[t.name] = template
	}
}

func RenderTestRun(testRun model.TestSuiteRun, w io.Writer) error {
	return templatesByName["test-run"].Execute(w, testRun)
}

func RenderTestRuns(testRuns []model.TestSuiteRun, w io.Writer) error {
	return templatesByName["test-runs"].Execute(w, testRuns)
}
