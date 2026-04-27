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
	analytics *handlers.AnalyticsHandler, admin *handlers.AdminHandler, clinic *handlers.ClinicHandler,
	mohDashboard *handlers.MOHDashboardHandler, mohReports *handlers.MOHReportsHandler) {

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
		usersGroup.GET("/me/assigned-area", middleware.RequireRole("phm"), users.GetMyAssignedArea)
		usersGroup.GET("/phm/assigned-areas", middleware.RequireRole("moh"), users.ListPHMAssignedAreas)
		usersGroup.PUT("/me", users.UpdateMe)
		usersGroup.PUT("/me/settings", users.UpdateSettings)
		usersGroup.POST("/request-mobile-change", users.RequestMobileChange)
		usersGroup.POST("/verify-mobile-change", users.VerifyMobileChange)
		usersGroup.POST("/phm", middleware.RequireRole("moh"), users.CreatePHM)
		usersGroup.GET("/:userId", middleware.RequireRole("phm"), users.GetParentByID)
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
		recGroup.GET("/due/phm", middleware.RequireRole("phm"), vaccRec.ListDueForPHM)
		recGroup.POST("/tracking", middleware.RequireRole("phm"), vaccRec.UpdateTracking)
		recGroup.PATCH("/child/:childId/next-due-date", middleware.RequireRole("phm", "moh"), vaccRec.UpdateNextDueDateByChildID)
		recGroup.GET("/:recordId", vaccRec.GetByID)
		recGroup.PUT("/:recordId", middleware.RequireRole("phm"), vaccRec.Update)
		recGroup.DELETE("/:recordId", middleware.RequireRole("moh"), vaccRec.Delete)
	}

	parentGroup := api.Group("/parent").Use(authMw).Use(middleware.RequireRole("parent"))
	{
		parentGroup.GET("/child/:child_id/vaccination-card", vaccRec.DownloadVaccinationCard)
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
		growthGroup.GET("/charts", growth.Charts)
		growthGroup.GET("/:childId/who-payload", growth.WHOByChildID)
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

	// Admin (admin-only)
	adminGroup := api.Group("/admin").Use(authMw).Use(middleware.RequireRole("admin"))
	{
		adminGroup.GET("/moh-accounts", admin.ListMOHAccounts)
		adminGroup.POST("/moh-accounts/request-otp", admin.RequestMOHAccountOTP)
		adminGroup.POST("/moh-accounts/complete", admin.CompleteMOHAccount)
		// NEW: Simplified single-step MOH account creation with temporary password
		adminGroup.POST("/moh-accounts/create", admin.CreateMOHAccount)
	}

	// Clinics
	clinicGroup := api.Group("/clinics").Use(authMw)
	{
		clinicGroup.POST("", middleware.RequireRole("phm"), clinic.CreateClinic)
		clinicGroup.GET("/my", middleware.RequireRole("phm"), clinic.ListMyClinics)
		clinicGroup.GET("/parent/due-vaccinations", middleware.RequireRole("parent"), clinic.ListParentDueVaccinations)
		clinicGroup.GET("/:clinicId", clinic.GetClinic)
		clinicGroup.GET("/:clinicId/due-children", clinic.GetDueChildren)
		clinicGroup.GET("/:clinicId/children", clinic.GetClinicChildren)
		clinicGroup.PUT("/:clinicId/status", middleware.RequireRole("phm", "moh"), clinic.UpdateClinicStatus)
		clinicGroup.POST("/:clinicId/attendance", middleware.RequireRole("phm", "moh"), clinic.UpdateAttendance)
	}

	// MOH Dashboard
	mohDashboardGroup := api.Group("/moh/dashboard").Use(authMw).Use(middleware.RequireRole("moh"))
	{
		mohDashboardGroup.GET("/total-children", mohDashboard.TotalChildren)
		mohDashboardGroup.GET("/gn-distribution", mohDashboard.ChildrenDistribution)
		mohDashboardGroup.GET("/coverage", mohDashboard.VaccinationCoverage)
		mohDashboardGroup.GET("/missed", mohDashboard.MissedVaccinations)
		mohDashboardGroup.GET("/phm-performance", mohDashboard.PHMPerformance)
		mohDashboardGroup.GET("/recent-children", mohDashboard.RecentChildren)
	}

	// MOH Reports
	mohReportsGroup := api.Group("/moh/reports").Use(authMw).Use(middleware.RequireRole("moh"))
	{
		mohReportsGroup.GET("/system-overview", mohReports.SystemOverviewReport)
		mohReportsGroup.GET("/coverage", mohReports.VaccinationCoverageReport)
		mohReportsGroup.GET("/missed", mohReports.MissedVaccinationReport)
		mohReportsGroup.GET("/phm-performance", mohReports.PHMPerformanceReport)
		mohReportsGroup.GET("/audit", mohReports.AuditReport)
		mohReportsGroup.GET("/:type/download", mohReports.DownloadReport)
		mohReportsGroup.GET("/:type/data", mohReports.GetReportData)
	}
}
