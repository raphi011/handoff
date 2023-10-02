package handoff

// WithServerPort sets the port that is used by the server.
// This can be overridden by flags.
func WithServerPort(port int) option {
	return func(s *Handoff) {
		s.config.Port = port
	}
}

// WithScheduledRun schedules a TestSuite to run at certain intervals.
// Ignored in CLI mode.
func WithScheduledRun(sr ScheduledRun) option {
	return func(s *Handoff) {
		s.schedules = append(s.schedules, sr)
	}
}

func WithTestSuiteFiles(files []string) option {
	return func(s *Handoff) {
		s.config.TestSuiteLibraryFiles = files
	}
}

func WithPlugin(p Plugin) option {
	return func(s *Handoff) {
		s.plugins.all = append(s.plugins.all, p)
	}
}

func WithTestSuite(suite TestSuite) option {
	return func(s *Handoff) {
		s._userProvidedTestSuites = append(s._userProvidedTestSuites, suite)
	}
}
