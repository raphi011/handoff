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
		err = body("").Render(templ.WithChildren(ctx, var_2), templBuffer)
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
		err = body("").Render(templ.WithChildren(ctx, var_7), templBuffer)
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
			_, err = templBuffer.WriteString("<main class=\"lg:pl-72\">")
			if err != nil {
				return err
			}
			err = component.Heading(tsr.SuiteName).Render(ctx, templBuffer)
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("<p>")
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
			_, err = templBuffer.WriteString("</p><p>")
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
			_, err = templBuffer.WriteString("</p>")
			if err != nil {
				return err
			}
			err = component.Stats().Render(ctx, templBuffer)
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("<h2 class=\"px-4 text-base/7 font-semibold text-white sm:px-6 lg:px-8\">")
			if err != nil {
				return err
			}
			var_19 := `Tests`
			_, err = templBuffer.WriteString(var_19)
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("</h2>")
			if err != nil {
				return err
			}
			err = component.TestRunTable(tsr).Render(ctx, templBuffer)
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("</main>")
			if err != nil {
				return err
			}
			if !templIsBuffer {
				_, err = io.Copy(w, templBuffer)
			}
			return err
		})
		err = body("").Render(templ.WithChildren(ctx, var_11), templBuffer)
		if err != nil {
			return err
		}
		if !templIsBuffer {
			_, err = templBuffer.WriteTo(w)
		}
		return err
	})
}

func RenderTestSuiteRuns(description string, runs []model.TestSuiteRun) templ.Component {
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
			err = component.SuiteRuns(description, runs).Render(ctx, templBuffer)
			if err != nil {
				return err
			}
			if !templIsBuffer {
				_, err = io.Copy(w, templBuffer)
			}
			return err
		})
		err = body("").Render(templ.WithChildren(ctx, var_21), templBuffer)
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
		var_22 := templ.GetChildren(ctx)
		if var_22 == nil {
			var_22 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		var_23 := templ.ComponentFunc(func(ctx context.Context, w io.Writer) (err error) {
			templBuffer, templIsBuffer := w.(*bytes.Buffer)
			if !templIsBuffer {
				templBuffer = templ.GetBuffer()
				defer templ.ReleaseBuffer(templBuffer)
			}
			err = component.TestSuites(suites).Render(ctx, templBuffer)
			if err != nil {
				return err
			}
			if !templIsBuffer {
				_, err = io.Copy(w, templBuffer)
			}
			return err
		})
		err = body("").Render(templ.WithChildren(ctx, var_23), templBuffer)
		if err != nil {
			return err
		}
		if !templIsBuffer {
			_, err = templBuffer.WriteTo(w)
		}
		return err
	})
}

func RenderTestSuitesWithRuns(suites []model.TestSuiteWithRuns) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) (err error) {
		templBuffer, templIsBuffer := w.(*bytes.Buffer)
		if !templIsBuffer {
			templBuffer = templ.GetBuffer()
			defer templ.ReleaseBuffer(templBuffer)
		}
		ctx = templ.InitializeContext(ctx)
		var_24 := templ.GetChildren(ctx)
		if var_24 == nil {
			var_24 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		var_25 := templ.ComponentFunc(func(ctx context.Context, w io.Writer) (err error) {
			templBuffer, templIsBuffer := w.(*bytes.Buffer)
			if !templIsBuffer {
				templBuffer = templ.GetBuffer()
				defer templ.ReleaseBuffer(templBuffer)
			}
			err = component.TestSuitesWithRuns(suites).Render(ctx, templBuffer)
			if err != nil {
				return err
			}
			if !templIsBuffer {
				_, err = io.Copy(w, templBuffer)
			}
			return err
		})
		err = body(" - Test Suites").Render(templ.WithChildren(ctx, var_25), templBuffer)
		if err != nil {
			return err
		}
		if !templIsBuffer {
			_, err = templBuffer.WriteTo(w)
		}
		return err
	})
}
