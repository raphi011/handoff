package handoff

// WithScheduledRun schedules a TestSuite to run at certain intervals.
func WithScheduledRun(sr ScheduledRun) option {
	return func(s *Server) {
		s.schedules = append(s.schedules, sr)
	}
}

func WithHook(p Hook) option {
	return func(s *Server) {
		s.hooks.all = append(s.hooks.all, p)
	}
}

func WithTestSuite(suite TestSuite) option {
	return func(s *Server) {
		s._userProvidedTestSuites = append(s._userProvidedTestSuites, suite)
	}
}
