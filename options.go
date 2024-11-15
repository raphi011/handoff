package handoff

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
