package handoff

// WithScheduledRun schedules a TestSuite to run at certain intervals.
// Ignored in CLI mode.
func WithScheduledRun(sr ScheduledRun) option {
	return func(s *Server) {
		s.schedules = append(s.schedules, sr)
	}
}

func WithTestSuiteFiles(files []string) option {
	return func(s *Server) {
		s.config.TestSuiteLibraryFiles = files
	}
}

func WithPlugin(p Plugin) option {
	return func(s *Server) {
		s.plugins.all = append(s.plugins.all, p)
	}
}

func WithTestSuite(suite TestSuite) option {
	return func(s *Server) {
		s._userProvidedTestSuites = append(s._userProvidedTestSuites, suite)
	}
}
