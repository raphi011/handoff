// Code generated by templ@v0.2.364 DO NOT EDIT.

package component

//lint:file-ignore SA4006 This context is only used if a nested component is present.

import "github.com/a-h/templ"
import "context"
import "io"
import "bytes"

func Heading(title string) templ.Component {
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
		_, err = templBuffer.WriteString("<div><div><nav class=\"sm:hidden\" aria-label=\"Back\"><a href=\"#\" class=\"flex items-center text-sm font-medium text-gray-500 hover:text-gray-700\"><svg class=\"-ml-1 mr-1 size-5 shrink-0 text-gray-400\" viewBox=\"0 0 20 20\" fill=\"currentColor\" aria-hidden=\"true\" data-slot=\"icon\"><path fill-rule=\"evenodd\" d=\"M11.78 5.22a.75.75 0 0 1 0 1.06L8.06 10l3.72 3.72a.75.75 0 1 1-1.06 1.06l-4.25-4.25a.75.75 0 0 1 0-1.06l4.25-4.25a.75.75 0 0 1 1.06 0Z\" clip-rule=\"evenodd\"></path></svg> ")
		if err != nil {
			return err
		}
		var_2 := `Back`
		_, err = templBuffer.WriteString(var_2)
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("</a></nav><nav class=\"hidden sm:flex\" aria-label=\"Breadcrumb\"><ol role=\"list\" class=\"flex items-center space-x-4\"><li><div class=\"flex\"><a href=\"#\" class=\"text-sm font-medium text-gray-500 hover:text-gray-700\">")
		if err != nil {
			return err
		}
		var_3 := `Jobs`
		_, err = templBuffer.WriteString(var_3)
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("</a></div></li><li><div class=\"flex items-center\"><svg class=\"size-5 shrink-0 text-gray-400\" viewBox=\"0 0 20 20\" fill=\"currentColor\" aria-hidden=\"true\" data-slot=\"icon\"><path fill-rule=\"evenodd\" d=\"M8.22 5.22a.75.75 0 0 1 1.06 0l4.25 4.25a.75.75 0 0 1 0 1.06l-4.25 4.25a.75.75 0 0 1-1.06-1.06L11.94 10 8.22 6.28a.75.75 0 0 1 0-1.06Z\" clip-rule=\"evenodd\"></path></svg><a href=\"#\" class=\"ml-4 text-sm font-medium text-gray-500 hover:text-gray-700\">")
		if err != nil {
			return err
		}
		var_4 := `Engineering`
		_, err = templBuffer.WriteString(var_4)
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("</a></div></li><li><div class=\"flex items-center\"><svg class=\"size-5 shrink-0 text-gray-400\" viewBox=\"0 0 20 20\" fill=\"currentColor\" aria-hidden=\"true\" data-slot=\"icon\"><path fill-rule=\"evenodd\" d=\"M8.22 5.22a.75.75 0 0 1 1.06 0l4.25 4.25a.75.75 0 0 1 0 1.06l-4.25 4.25a.75.75 0 0 1-1.06-1.06L11.94 10 8.22 6.28a.75.75 0 0 1 0-1.06Z\" clip-rule=\"evenodd\"></path></svg><a href=\"#\" aria-current=\"page\" class=\"ml-4 text-sm font-medium text-gray-500 hover:text-gray-700\">")
		if err != nil {
			return err
		}
		var_5 := `Back End Developer`
		_, err = templBuffer.WriteString(var_5)
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("</a></div></li></ol></nav></div><div class=\"mt-2 md:flex md:items-center md:justify-between\"><div class=\"min-w-0 flex-1\"><h2 class=\"text-2xl/7 font-bold text-gray-900 sm:truncate sm:text-3xl sm:tracking-tight\">")
		if err != nil {
			return err
		}
		var var_6 string = title
		_, err = templBuffer.WriteString(templ.EscapeString(var_6))
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("</h2></div><div class=\"mt-4 flex shrink-0 md:ml-4 md:mt-0\"><button type=\"button\" class=\"inline-flex items-center rounded-md bg-white px-3 py-2 text-sm font-semibold text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 hover:bg-gray-50\">")
		if err != nil {
			return err
		}
		var_7 := `Edit`
		_, err = templBuffer.WriteString(var_7)
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("</button><button type=\"button\" class=\"ml-3 inline-flex items-center rounded-md bg-indigo-600 px-3 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-600\">")
		if err != nil {
			return err
		}
		var_8 := `Publish`
		_, err = templBuffer.WriteString(var_8)
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("</button></div></div></div>")
		if err != nil {
			return err
		}
		if !templIsBuffer {
			_, err = templBuffer.WriteTo(w)
		}
		return err
	})
}
