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
			auth.POST("/forgot-password", s.authHandler.ForgotPassword())
			auth.POST("/reset-password", s.authHandler.ResetPassword())
			auth.POST("/logout", s.JWTMiddleware(), s.authHandler.Logout())
		}

		users := v1.Group("/users", s.JWTMiddleware())
		{
			users.GET("/me", s.userHandler.GetMe())
			users.PATCH("/me", s.userHandler.UpdateMe())
			users.PATCH("/me/password", s.userHandler.ChangePassword())
		}

		videos := v1.Group("/videos", s.JWTMiddleware())
		{
			videos.GET("", s.videoHandler.List())
			videos.POST("/upload", s.videoHandler.Upload())
			videos.GET("/:id/download", s.videoHandler.GetDownload())
			videos.GET("/:id/status", s.videoHandler.GetStatus())
			videos.DELETE("/:id", s.videoHandler.Delete())
		}

		wsGroup := v1.Group("/ws")
		{
			wsGroup.GET("/pipeline", s.pipelineWS.HandlePipeline())
		}
	}
}
