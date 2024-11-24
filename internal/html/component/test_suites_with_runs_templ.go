// Code generated by templ - DO NOT EDIT.

// templ: version: v0.2.793
package component

//lint:file-ignore SA4006 This context is only used if a nested component is present.

import "github.com/a-h/templ"
import templruntime "github.com/a-h/templ/runtime"

import (
	"fmt"
	"github.com/raphi011/handoff/internal/model"
	"sort"
)

// merge runs and sort them by time
func mergeRuns(suitesWithRuns []model.TestSuiteWithRuns) []model.TestSuiteRun {
	var runs []model.TestSuiteRun
	for _, suite := range suitesWithRuns {
		runs = append(runs, suite.SuiteRuns...)
	}
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].Start.After(runs[j].Start)
	})
	return runs
}

func TestSuitesWithRuns(suitesWithRuns []model.TestSuiteWithRuns) templ.Component {
	return templruntime.GeneratedTemplate(func(templ_7745c5c3_Input templruntime.GeneratedComponentInput) (templ_7745c5c3_Err error) {
		templ_7745c5c3_W, ctx := templ_7745c5c3_Input.Writer, templ_7745c5c3_Input.Context
		if templ_7745c5c3_CtxErr := ctx.Err(); templ_7745c5c3_CtxErr != nil {
			return templ_7745c5c3_CtxErr
		}
		templ_7745c5c3_Buffer, templ_7745c5c3_IsBuffer := templruntime.GetBuffer(templ_7745c5c3_W)
		if !templ_7745c5c3_IsBuffer {
			defer func() {
				templ_7745c5c3_BufErr := templruntime.ReleaseBuffer(templ_7745c5c3_Buffer)
				if templ_7745c5c3_Err == nil {
					templ_7745c5c3_Err = templ_7745c5c3_BufErr
				}
			}()
		}
		ctx = templ.InitializeContext(ctx)
		templ_7745c5c3_Var1 := templ.GetChildren(ctx)
		if templ_7745c5c3_Var1 == nil {
			templ_7745c5c3_Var1 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<main class=\"lg:pl-72\"><header class=\"flex items-center justify-between border-b border-white/5 px-4 py-4 sm:px-6 sm:py-6 lg:px-8\"><h1 class=\"text-base/7 font-semibold text-gray-900\">Test Suites</h1></header><ul role=\"list\" class=\"divide-y divide-gray/5 border-t border-b\">")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		for _, suite := range suitesWithRuns {
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<li class=\"relative flex items-center space-x-4 px-4 py-4 sm:px-6 lg:px-8\"><div class=\"min-w-0 flex-auto\"><div class=\"flex items-center gap-x-3\"><div class=\"flex-none rounded-full bg-gray-600/10 p-1 text-gray-500\"><div class=\"size-2 rounded-full bg-current\"></div></div><h2 class=\"min-w-0 text-sm/6 font-semibold text-gray-500\"><a href=\"")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			var templ_7745c5c3_Var2 templ.SafeURL = templ.URL(fmt.Sprintf("/suites/%s/runs", suite.Suite.Name))
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(string(templ_7745c5c3_Var2)))
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("\" class=\"flex gap-x-2\"><span class=\"truncate\">Staging</span> <span class=\"text-gray-400\">/</span> <span class=\"whitespace-nowrap text-gray-900\">")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			var templ_7745c5c3_Var3 string
			templ_7745c5c3_Var3, templ_7745c5c3_Err = templ.JoinStringErrs(suite.Suite.Name)
			if templ_7745c5c3_Err != nil {
				return templ.Error{Err: templ_7745c5c3_Err, FileName: `internal/html/component/test_suites_with_runs.templ`, Line: 38, Col: 73}
			}
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var3))
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("</span> <span class=\"absolute inset-0\"></span></a></h2></div><div class=\"mt-3 flex items-center gap-x-2.5 text-xs/5 text-gray-400\"><p class=\"truncate\">")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			var templ_7745c5c3_Var4 string
			templ_7745c5c3_Var4, templ_7745c5c3_Err = templ.JoinStringErrs(fmt.Sprintf("%d tests in suite", len(suite.Suite.Tests)))
			if templ_7745c5c3_Err != nil {
				return templ.Error{Err: templ_7745c5c3_Err, FileName: `internal/html/component/test_suites_with_runs.templ`, Line: 44, Col: 85}
			}
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var4))
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("</p><svg viewBox=\"0 0 2 2\" class=\"size-0.5 flex-none fill-gray-300\"><circle cx=\"1\" cy=\"1\" r=\"1\"></circle></svg><p class=\"whitespace-nowrap\">Initiated 1m 32s ago</p></div></div><!-- <div class=\"flex-none rounded-full bg-gray-400/10 px-2 py-1 text-xs font-medium text-gray-400 ring-1 ring-inset ring-gray-400/20\">Preview</div> --><!-- <svg class=\"size-5 flex-none text-gray-400\" viewBox=\"0 0 20 20\" fill=\"currentColor\" aria-hidden=\"true\" data-slot=\"icon\"> --><!-- \t<path fill-rule=\"evenodd\" d=\"M8.22 5.22a.75.75 0 0 1 1.06 0l4.25 4.25a.75.75 0 0 1 0 1.06l-4.25 4.25a.75.75 0 0 1-1.06-1.06L11.94 10 8.22 6.28a.75.75 0 0 1 0-1.06Z\" clip-rule=\"evenodd\"></path> --><!-- </svg> --></li>")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("</ul></main><aside class=\"bg-white lg:fixed lg:bottom-0 lg:right-0 lg:top-0 lg:w-2/6 lg:overflow-y-auto lg:border-l lg:border-gray/5\"><header class=\"flex items-center justify-between border-b border-white/5 px-4 py-4 sm:px-6 sm:py-6 lg:px-8\"><h2 class=\"text-base/7 font-semibold text-gray-900\">Activity</h2><a href=\"#\" class=\"text-sm/6 font-semibold text-indigo-400\">View all</a></header><div class=\"divide-y divide-white/5 border-t p-5\">")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		templ_7745c5c3_Err = RunActivity(mergeRuns(suitesWithRuns)).Render(ctx, templ_7745c5c3_Buffer)
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("</div></aside>")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		return templ_7745c5c3_Err
	})
}

var _ = templruntime.GeneratedTemplate
