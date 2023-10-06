package packagetestexample

import "github.com/raphi011/handoff"

func init() {
	ts := []handoff.TestSuite{
		{
			Name:      "external-suite-succeed",
			Namespace: "",
			Tests: []handoff.TestFunc{
				Succeed,
			},
		},
	}

	handoff.Register(ts, nil)
}

func Succeed(t handoff.TB) {

}
