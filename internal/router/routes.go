package router

import (
	"ncvms/internal/handlers"
	"ncvms/internal/middleware"

	"github.com/gin-gonic/gin"
)

func Setup(engine *gin.Engine, jwtSecret string, auth *handlers.AuthHandler, users *handlers.UsersHandler,
	children *handlers.ChildrenHandler, vaccines *handlers.VaccinesHandler,
	vaccRec *handlers.VaccinationRecordsHandler, sched *handlers.SchedulesHandler, growth *handlers.GrowthHandler,
	notif *handlers.NotificationsHandler, reports *handlers.ReportsHandler, audit *handlers.AuditHandler,
	analytics *handlers.AnalyticsHandler) {

	api := engine.Group("/api/v1")
	authMw := middleware.AuthRequired(jwtSecret)

	// Public auth
	authGroup := api.Group("/auth")
	{
		authGroup.POST("/login", auth.Login)
		authGroup.POST("/register", auth.Register)
		authGroup.POST("/forgot-password", auth.ForgotPassword)
		authGroup.POST("/reset-password", auth.ResetPassword)
	}
	api.POST("/auth/logout", authMw, auth.Logout)
	api.POST("/auth/change-password", authMw, auth.ChangePassword)

	// Users (authenticated)
	usersGroup := api.Group("/users").Use(authMw)
	{
		usersGroup.GET("/me", users.GetMe)
		usersGroup.PUT("/me", users.UpdateMe)
		usersGroup.PUT("/me/settings", users.UpdateSettings)
		usersGroup.POST("/request-mobile-change", users.RequestMobileChange)
		usersGroup.POST("/verify-mobile-change", users.VerifyMobileChange)
		usersGroup.POST("/phm", middleware.RequireRole("moh"), users.CreatePHM)
	}

	// Children
	childrenGroup := api.Group("/children").Use(authMw)
	{
		childrenGroup.POST("", middleware.RequireRole("phm"), children.Register)
		childrenGroup.GET("", children.List)
		childrenGroup.GET("/my", middleware.RequireRole("phm"), children.ListMy)
		childrenGroup.GET("/search", children.Search)
		childrenGroup.GET("/:childId", children.GetByID)
		childrenGroup.PUT("/:childId", middleware.RequireRole("phm", "moh"), children.Update)
		childrenGroup.POST("/:childId/link-parent/otp/request", middleware.RequireRole("parent"), children.RequestLinkOTP)
		// Backward-compatible alias for clients expecting a dedicated OTP verification path.
		childrenGroup.POST("/:childId/link-parent/otp/verify", middleware.RequireRole("parent"), children.LinkParent)
		childrenGroup.POST("/:childId/link-parent", middleware.RequireRole("parent"), children.LinkParent)
	}

	// Vaccines
	vaccinesGroup := api.Group("/vaccines").Use(authMw)
	{
		vaccinesGroup.GET("", vaccines.List)
		vaccinesGroup.GET("/:vaccineId", vaccines.GetByID)
	}

	// Vaccination records
	recGroup := api.Group("/vaccination-records").Use(authMw)
	{
		recGroup.POST("", middleware.RequireRole("phm"), vaccRec.Create)
		recGroup.GET("", vaccRec.List)
		recGroup.GET("/:recordId", vaccRec.GetByID)
		recGroup.PUT("/:recordId", middleware.RequireRole("phm"), vaccRec.Update)
		recGroup.DELETE("/:recordId", middleware.RequireRole("moh"), vaccRec.Delete)
	}

	// Schedules
	schedGroup := api.Group("/schedules").Use(authMw)
	{
		schedGroup.GET("", sched.List)
		schedGroup.POST("", middleware.RequireRole("phm"), sched.Create)
		schedGroup.PUT("/:scheduleId/status", middleware.RequireRole("phm", "moh"), sched.UpdateStatus)
		schedGroup.POST("/:scheduleId/send-reminder", middleware.RequireRole("phm", "moh"), sched.SendReminder)
	}

	// Growth records
	growthGroup := api.Group("/growth-records").Use(authMw)
	{
		growthGroup.POST("", middleware.RequireRole("phm"), growth.Create)
		growthGroup.GET("", growth.List)
	}

	// Notifications
	notifGroup := api.Group("/notifications").Use(authMw)
	{
		notifGroup.GET("", notif.List)
		notifGroup.POST("", middleware.RequireRole("phm", "moh"), notif.Create)
		notifGroup.PUT("/read-all", notif.MarkAllRead)
		notifGroup.PUT("/:notificationId/read", notif.MarkRead)
	}

	// Reports (MOH)
	reportsGroup := api.Group("/reports").Use(authMw).Use(middleware.RequireRole("moh"))
	{
		reportsGroup.POST("/generate", reports.Generate)
		reportsGroup.GET("", reports.List)
		reportsGroup.GET("/:reportId/download", reports.Download)
	}

	// Audit logs (MOH)
	auditGroup := api.Group("/audit-logs").Use(authMw).Use(middleware.RequireRole("moh"))
	{
		auditGroup.GET("", audit.List)
		auditGroup.GET("/export", audit.Export)
	}

	// Analytics
	analyticsGroup := api.Group("/analytics").Use(authMw)
	{
		analyticsGroup.GET("/dashboard", middleware.RequireRole("moh"), analytics.MOHDashboard)
		analyticsGroup.GET("/vaccination-coverage", middleware.RequireRole("moh"), analytics.VaccinationCoverage)
		analyticsGroup.GET("/area-performance", middleware.RequireRole("moh"), analytics.AreaPerformance)
		analyticsGroup.GET("/phm-dashboard", middleware.RequireRole("phm"), analytics.PHMDashboard)
		analyticsGroup.GET("/parent-dashboard", middleware.RequireRole("parent"), analytics.ParentDashboard)
	}
}
