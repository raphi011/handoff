package handoff

// WithScheduledRun schedules a TestSuite to run at certain intervals.
func WithScheduledRun(sr ScheduledRun) Option {
	return func(s *Server) {
		s.schedules = append(s.schedules, sr)
	}
}

func WithHook(p Hook) Option {
	return func(s *Server) {
		s.hooks.all = append(s.hooks.all, p)
	}
}

func WithTestSuite(suite TestSuite) Option {
	return func(s *Server) {
		s._userProvidedTestSuites = append(s._userProvidedTestSuites, suite)
	}
}
