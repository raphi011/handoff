// Code generated by templ@v0.2.364 DO NOT EDIT.

package html

//lint:file-ignore SA4006 This context is only used if a nested component is present.

import "github.com/a-h/templ"
import "context"
import "io"
import "bytes"

import (
	"fmt"

	"github.com/raphi011/handoff/internal/html/component"
	"github.com/raphi011/handoff/internal/model"
)

func RenderTestRun(tr model.TestRun) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) (err error) {
		templBuffer, templIsBuffer := w.(*bytes.Buffer)
		if !templIsBuffer {
			templBuffer = templ.GetBuffer()
			defer templ.ReleaseBuffer(templBuffer)
		}
		ctx = templ.InitializeContext(ctx)
		var_1 := templ.GetChildren(ctx)
		if var_1 == nil {
			var_1 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		var_2 := templ.ComponentFunc(func(ctx context.Context, w io.Writer) (err error) {
			templBuffer, templIsBuffer := w.(*bytes.Buffer)
			if !templIsBuffer {
				templBuffer = templ.GetBuffer()
				defer templ.ReleaseBuffer(templBuffer)
			}
			_, err = templBuffer.WriteString("<h1>")
			if err != nil {
				return err
			}
			var var_3 string = tr.Name
			_, err = templBuffer.WriteString(templ.EscapeString(var_3))
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("</h1> <h2>")
			if err != nil {
				return err
			}
			var_4 := `Logs`
			_, err = templBuffer.WriteString(var_4)
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("</h2> <code>")
			if err != nil {
				return err
			}
			var var_5 string = tr.Logs
			_, err = templBuffer.WriteString(templ.EscapeString(var_5))
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("</code>")
			if err != nil {
				return err
			}
			if !templIsBuffer {
				_, err = io.Copy(w, templBuffer)
			}
			return err
		})
		err = body().Render(templ.WithChildren(ctx, var_2), templBuffer)
		if err != nil {
			return err
		}
		if !templIsBuffer {
			_, err = templBuffer.WriteTo(w)
		}
		return err
	})
}

func RenderSchedules(schedules []model.ScheduledRun) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) (err error) {
		templBuffer, templIsBuffer := w.(*bytes.Buffer)
		if !templIsBuffer {
			templBuffer = templ.GetBuffer()
			defer templ.ReleaseBuffer(templBuffer)
		}
		ctx = templ.InitializeContext(ctx)
		var_6 := templ.GetChildren(ctx)
		if var_6 == nil {
			var_6 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		var_7 := templ.ComponentFunc(func(ctx context.Context, w io.Writer) (err error) {
			templBuffer, templIsBuffer := w.(*bytes.Buffer)
			if !templIsBuffer {
				templBuffer = templ.GetBuffer()
				defer templ.ReleaseBuffer(templBuffer)
			}
			_, err = templBuffer.WriteString("<h2>")
			if err != nil {
				return err
			}
			var_8 := `Scheduled runs`
			_, err = templBuffer.WriteString(var_8)
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("</h2> <ul>")
			if err != nil {
				return err
			}
			for _, s := range schedules {
				_, err = templBuffer.WriteString("<li>")
				if err != nil {
					return err
				}
				var var_9 string = s.Name
				_, err = templBuffer.WriteString(templ.EscapeString(var_9))
				if err != nil {
					return err
				}
				_, err = templBuffer.WriteString("</li>")
				if err != nil {
					return err
				}
			}
			_, err = templBuffer.WriteString("</ul>")
			if err != nil {
				return err
			}
			if !templIsBuffer {
				_, err = io.Copy(w, templBuffer)
			}
			return err
		})
		err = body().Render(templ.WithChildren(ctx, var_7), templBuffer)
		if err != nil {
			return err
		}
		if !templIsBuffer {
			_, err = templBuffer.WriteTo(w)
		}
		return err
	})
}

func RenderTestSuiteRun(tsr model.TestSuiteRun) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) (err error) {
		templBuffer, templIsBuffer := w.(*bytes.Buffer)
		if !templIsBuffer {
			templBuffer = templ.GetBuffer()
			defer templ.ReleaseBuffer(templBuffer)
		}
		ctx = templ.InitializeContext(ctx)
		var_10 := templ.GetChildren(ctx)
		if var_10 == nil {
			var_10 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		var_11 := templ.ComponentFunc(func(ctx context.Context, w io.Writer) (err error) {
			templBuffer, templIsBuffer := w.(*bytes.Buffer)
			if !templIsBuffer {
				templBuffer = templ.GetBuffer()
				defer templ.ReleaseBuffer(templBuffer)
			}
			err = component.Heading(tsr.SuiteName).Render(ctx, templBuffer)
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString(" <p>")
			if err != nil {
				return err
			}
			var_12 := `Started at `
			_, err = templBuffer.WriteString(var_12)
			if err != nil {
				return err
			}
			var var_13 string = tsr.Start.Format("02.01 15:04:05")
			_, err = templBuffer.WriteString(templ.EscapeString(var_13))
			if err != nil {
				return err
			}
			var_14 := `, took `
			_, err = templBuffer.WriteString(var_14)
			if err != nil {
				return err
			}
			var var_15 string = fmt.Sprintf("%d", tsr.DurationInMS)
			_, err = templBuffer.WriteString(templ.EscapeString(var_15))
			if err != nil {
				return err
			}
			var_16 := `ms to finish.`
			_, err = templBuffer.WriteString(var_16)
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("</p> <p>")
			if err != nil {
				return err
			}
			var_17 := `Is flaky: `
			_, err = templBuffer.WriteString(var_17)
			if err != nil {
				return err
			}
			var var_18 string = fmt.Sprintf("%t", tsr.Flaky)
			_, err = templBuffer.WriteString(templ.EscapeString(var_18))
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("</p> ")
			if err != nil {
				return err
			}
			err = component.Stats().Render(ctx, templBuffer)
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString(" <h2 class=\"px-4 text-base/7 font-semibold text-white sm:px-6 lg:px-8\">")
			if err != nil {
				return err
			}
			var_19 := `Tests`
			_, err = templBuffer.WriteString(var_19)
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("</h2> ")
			if err != nil {
				return err
			}
			err = component.TestRunTable(tsr).Render(ctx, templBuffer)
			if err != nil {
				return err
			}
			if !templIsBuffer {
				_, err = io.Copy(w, templBuffer)
			}
			return err
		})
		err = body().Render(templ.WithChildren(ctx, var_11), templBuffer)
		if err != nil {
			return err
		}
		if !templIsBuffer {
			_, err = templBuffer.WriteTo(w)
		}
		return err
	})
}

func RenderTestSuiteRuns(runs []model.TestSuiteRun) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) (err error) {
		templBuffer, templIsBuffer := w.(*bytes.Buffer)
		if !templIsBuffer {
			templBuffer = templ.GetBuffer()
			defer templ.ReleaseBuffer(templBuffer)
		}
		ctx = templ.InitializeContext(ctx)
		var_20 := templ.GetChildren(ctx)
		if var_20 == nil {
			var_20 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		var_21 := templ.ComponentFunc(func(ctx context.Context, w io.Writer) (err error) {
			templBuffer, templIsBuffer := w.(*bytes.Buffer)
			if !templIsBuffer {
				templBuffer = templ.GetBuffer()
				defer templ.ReleaseBuffer(templBuffer)
			}
			_, err = templBuffer.WriteString("<h2>")
			if err != nil {
				return err
			}
			var_22 := `Testruns`
			_, err = templBuffer.WriteString(var_22)
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("</h2> <ul>")
			if err != nil {
				return err
			}
			for _, tsr := range runs {
				_, err = templBuffer.WriteString("<li><a href=\"")
				if err != nil {
					return err
				}
				var var_23 templ.SafeURL = templ.URL(fmt.Sprintf("/suites/%s/runs/%d", tsr.SuiteName, tsr.ID))
				_, err = templBuffer.WriteString(templ.EscapeString(string(var_23)))
				if err != nil {
					return err
				}
				_, err = templBuffer.WriteString("\">")
				if err != nil {
					return err
				}
				var var_24 string = tsr.SuiteName
				_, err = templBuffer.WriteString(templ.EscapeString(var_24))
				if err != nil {
					return err
				}
				_, err = templBuffer.WriteString(" ")
				if err != nil {
					return err
				}
				var_25 := `(#`
				_, err = templBuffer.WriteString(var_25)
				if err != nil {
					return err
				}
				var var_26 string = fmt.Sprintf("%d", tsr.ID)
				_, err = templBuffer.WriteString(templ.EscapeString(var_26))
				if err != nil {
					return err
				}
				var_27 := `): `
				_, err = templBuffer.WriteString(var_27)
				if err != nil {
					return err
				}
				var var_28 string = string(tsr.Result)
				_, err = templBuffer.WriteString(templ.EscapeString(var_28))
				if err != nil {
					return err
				}
				_, err = templBuffer.WriteString("</a></li>")
				if err != nil {
					return err
				}
			}
			_, err = templBuffer.WriteString("</ul>")
			if err != nil {
				return err
			}
			if !templIsBuffer {
				_, err = io.Copy(w, templBuffer)
			}
			return err
		})
		err = body().Render(templ.WithChildren(ctx, var_21), templBuffer)
		if err != nil {
			return err
		}
		if !templIsBuffer {
			_, err = templBuffer.WriteTo(w)
		}
		return err
	})
}

func RenderTestSuites(suites []model.TestSuite) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) (err error) {
		templBuffer, templIsBuffer := w.(*bytes.Buffer)
		if !templIsBuffer {
			templBuffer = templ.GetBuffer()
			defer templ.ReleaseBuffer(templBuffer)
		}
		ctx = templ.InitializeContext(ctx)
		var_29 := templ.GetChildren(ctx)
		if var_29 == nil {
			var_29 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		var_30 := templ.ComponentFunc(func(ctx context.Context, w io.Writer) (err error) {
			templBuffer, templIsBuffer := w.(*bytes.Buffer)
			if !templIsBuffer {
				templBuffer = templ.GetBuffer()
				defer templ.ReleaseBuffer(templBuffer)
			}
			_, err = templBuffer.WriteString("<h2>")
			if err != nil {
				return err
			}
			var_31 := `Test suites`
			_, err = templBuffer.WriteString(var_31)
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("</h2> <ul>")
			if err != nil {
				return err
			}
			for _, ts := range suites {
				_, err = templBuffer.WriteString("<li><a href=\"")
				if err != nil {
					return err
				}
				var var_32 templ.SafeURL = templ.URL(fmt.Sprintf("/suites/%s/runs", ts.Name))
				_, err = templBuffer.WriteString(templ.EscapeString(string(var_32)))
				if err != nil {
					return err
				}
				_, err = templBuffer.WriteString("\">")
				if err != nil {
					return err
				}
				var var_33 string = ts.Name
				_, err = templBuffer.WriteString(templ.EscapeString(var_33))
				if err != nil {
					return err
				}
				_, err = templBuffer.WriteString("</a></li>")
				if err != nil {
					return err
				}
			}
			_, err = templBuffer.WriteString("</ul>")
			if err != nil {
				return err
			}
			if !templIsBuffer {
				_, err = io.Copy(w, templBuffer)
			}
			return err
		})
		err = body().Render(templ.WithChildren(ctx, var_30), templBuffer)
		if err != nil {
			return err
		}
		if !templIsBuffer {
			_, err = templBuffer.WriteTo(w)
		}
		return err
	})
}
