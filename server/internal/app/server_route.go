package app

func (s *Server) registerRoutes() {
	v1 := s.router.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", s.authHandler.Register())
			auth.POST("/login", s.authHandler.Login())
			auth.POST("/google", s.authHandler.LoginWithGoogle())
			auth.POST("/refresh", s.authHandler.RefreshToken())
			auth.POST("/logout", s.JWTMiddleware(), s.authHandler.Logout())
		}

		videos := v1.Group("/videos")
		{
			videos.POST("/upload", s.uploadHandler.UploadVideo())
		}
	}
}
