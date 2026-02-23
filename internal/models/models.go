package models

import "time"

type User struct {
	UserId             string                 `json:"userId"`
	Email              string                 `json:"email"`
	NIC                string                 `json:"nic"`
	Role               string                 `json:"role"`
	Name               string                 `json:"name"`
	PhoneNumber        string                 `json:"phoneNumber"`
	Address            string                 `json:"address"`
	LanguagePreference string                 `json:"languagePreference"`
	NotificationSettings map[string]interface{} `json:"notifications,omitempty"`
	CreatedAt          time.Time              `json:"createdAt"`
}

type UserWithPassword struct {
	User
	PasswordHash string
	AreaCode     *string
	UpdatedAt    time.Time
}

type Child struct {
	ChildId             string   `json:"childId"`
	RegistrationNumber  string   `json:"registrationNumber"`
	FirstName           string   `json:"firstName"`
	LastName            string   `json:"lastName"`
	DateOfBirth         string   `json:"dateOfBirth"`
	Gender              string   `json:"gender"`
	BloodGroup          string   `json:"bloodGroup"`
	BirthWeight         *float64 `json:"birthWeight"`
	BirthHeight         *float64 `json:"birthHeight"`
	HeadCircumference   *float64 `json:"headCircumference"`
	ParentId            *string  `json:"parentId"`
	RegisteredBy        *string  `json:"registeredBy"`
	AreaCode            string   `json:"areaCode"`
	AreaName            string   `json:"areaName"`
	VaccinationStatus   string   `json:"vaccinationStatus,omitempty"`
	CreatedAt           time.Time `json:"createdAt"`
}

type ChildDetail struct {
	Child
	MotherName   string  `json:"motherName"`
	MotherNIC    string  `json:"motherNic"`
	FatherName   string  `json:"fatherName"`
	FatherNIC    string  `json:"fatherNic"`
	District     string  `json:"district"`
	DsDivision   string  `json:"dsDivision"`
	GnDivision   string  `json:"gnDivision"`
	Address      string  `json:"address"`
}

type Vaccine struct {
	VaccineId        string `json:"vaccineId"`
	Name             string `json:"name"`
	Manufacturer     string `json:"manufacturer"`
	DosageInfo       string `json:"dosageInfo"`
	RecommendedAge   int    `json:"recommendedAge"`
	IntervalDays     int    `json:"intervalDays"`
	Description      string `json:"description"`
	IsActive         bool   `json:"isActive"`
}

type VaccinationRecord struct {
	RecordId         string    `json:"recordId"`
	ChildId          string    `json:"childId"`
	VaccineId        string    `json:"vaccineId"`
	VaccineName      string    `json:"vaccineName"`
	AdministeredDate string    `json:"administeredDate"`
	BatchNumber      string    `json:"batchNumber"`
	AdministeredBy   string    `json:"administeredBy"`
	Location         string    `json:"location"`
	Site             string    `json:"site"`
	DoseNumber       *int      `json:"doseNumber"`
	NextDueDate      *string   `json:"nextDueDate"`
	Status           string    `json:"status"`
	Notes            string    `json:"notes"`
	CreatedAt        time.Time `json:"createdAt"`
}

type VaccinationSchedule struct {
	ScheduleId    string `json:"scheduleId"`
	ChildId       string `json:"childId"`
	VaccineId     string `json:"vaccineId"`
	VaccineName   string `json:"vaccineName"`
	ScheduledDate string `json:"scheduledDate"`
	DueDate       string `json:"dueDate"`
	Status        string `json:"status"`
	ReminderSent  bool   `json:"reminderSent"`
}

type GrowthRecord struct {
	RecordId           string    `json:"recordId"`
	ChildId            string    `json:"childId"`
	RecordedDate       string    `json:"recordedDate"`
	Height             *float64  `json:"height"`
	Weight             *float64  `json:"weight"`
	HeadCircumference  *float64  `json:"headCircumference"`
	RecordedBy         string    `json:"recordedBy"`
	Notes              string    `json:"notes"`
	CreatedAt          time.Time `json:"createdAt"`
}

type Notification struct {
	NotificationId   string    `json:"notificationId"`
	RecipientId      string    `json:"recipientId"`
	Type             string    `json:"type"`
	Message          string    `json:"message"`
	RelatedChildId   *string   `json:"relatedChildId"`
	SentDate         time.Time `json:"sentDate"`
	IsRead           bool      `json:"isRead"`
}

type Report struct {
	ReportId       string    `json:"reportId"`
	ReportType     string    `json:"reportType"`
	GeneratedBy    string    `json:"generatedBy"`
	GeneratedDate  time.Time `json:"generatedDate"`
	StartDate      string    `json:"startDate"`
	EndDate        string    `json:"endDate"`
	DownloadUrl    string    `json:"downloadUrl"`
}

type AuditLog struct {
	LogId       string    `json:"logId"`
	UserId      *string   `json:"userId"`
	UserRole    string    `json:"userRole"`
	UserName    string    `json:"userName"`
	Action      string    `json:"action"`
	EntityType  string    `json:"entityType"`
	EntityId    string    `json:"entityId"`
	Details     string    `json:"details"`
	Timestamp   time.Time `json:"timestamp"`
	IpAddress   string    `json:"ipAddress"`
}

