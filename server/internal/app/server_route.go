package app

func (s *Server) registerRoutes() {
	v1 := s.router.Group("/api/v1")
	{
		videos := v1.Group("/videos")
		{
			videos.POST("/upload", s.uploadHandler.UploadVideo())
		}
	}
}
