// Code generated by templ - DO NOT EDIT.

// templ: version: v0.2.793
package component

//lint:file-ignore SA4006 This context is only used if a nested component is present.

import "github.com/a-h/templ"
import templruntime "github.com/a-h/templ/runtime"

func RunActivity() templ.Component {
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
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<div class=\"flow-root\"><ul role=\"list\" class=\"-mb-8\"><li><div class=\"relative pb-8\"><span class=\"absolute left-4 top-4 -ml-px h-full w-0.5 bg-gray-200\" aria-hidden=\"true\"></span><div class=\"relative flex space-x-3\"><div><span class=\"flex size-8 items-center justify-center rounded-full bg-green-500 ring-8 ring-white\"><svg class=\"size-5 text-white\" viewBox=\"0 0 20 20\" fill=\"currentColor\" aria-hidden=\"true\" data-slot=\"icon\"><path fill-rule=\"evenodd\" d=\"M16.704 4.153a.75.75 0 0 1 .143 1.052l-8 10.5a.75.75 0 0 1-1.127.075l-4.5-4.5a.75.75 0 0 1 1.06-1.06l3.894 3.893 7.48-9.817a.75.75 0 0 1 1.05-.143Z\" clip-rule=\"evenodd\"></path></svg></span></div><div class=\"flex min-w-0 flex-1 justify-between space-x-4 pt-1.5\"><div><p class=\"text-sm text-gray-500\">Test run 2 1 passed for <a href=\"#\" class=\"font-medium text-gray-900\">external-suite-succeed</a></p></div><div class=\"whitespace-nowrap text-right text-sm text-gray-500\"><time datetime=\"2020-09-28\">1 min ago</time></div></div></div></div></li><li><div class=\"relative pb-8\"><span class=\"absolute left-4 top-4 -ml-px h-full w-0.5 bg-gray-200\" aria-hidden=\"true\"></span><div class=\"relative flex space-x-3\"><div><span class=\"flex size-8 items-center justify-center rounded-full bg-green-500 ring-8 ring-white\"><svg class=\"size-5 text-white\" viewBox=\"0 0 20 20\" fill=\"currentColor\" aria-hidden=\"true\" data-slot=\"icon\"><path fill-rule=\"evenodd\" d=\"M16.704 4.153a.75.75 0 0 1 .143 1.052l-8 10.5a.75.75 0 0 1-1.127.075l-4.5-4.5a.75.75 0 0 1 1.06-1.06l3.894 3.893 7.48-9.817a.75.75 0 0 1 1.05-.143Z\" clip-rule=\"evenodd\"></path></svg></span></div><div class=\"flex min-w-0 flex-1 justify-between space-x-4 pt-1.5\"><div><p class=\"text-sm text-gray-500\">Test run 1 passed for <a href=\"#\" class=\"font-medium text-gray-900\">external-suite-succeed</a></p></div><div class=\"whitespace-nowrap text-right text-sm text-gray-500\"><time datetime=\"2020-09-28\">2 min ago</time></div></div></div></div></li></ul></div>")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		return templ_7745c5c3_Err
	})
}

var _ = templruntime.GeneratedTemplate
