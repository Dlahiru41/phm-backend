package models

import "time"

type User struct {
	UserId               string                 `json:"userId"`
	Email                string                 `json:"email"`
	NIC                  string                 `json:"nic"`
	Role                 string                 `json:"role"`
	Name                 string                 `json:"name"`
	PhoneNumber          string                 `json:"phoneNumber"`
	Address              string                 `json:"address"`
	LanguagePreference   string                 `json:"languagePreference"`
	NotificationSettings map[string]interface{} `json:"notifications,omitempty"`
	CreatedAt            time.Time              `json:"createdAt"`
}

type UserWithPassword struct {
	User
	PasswordHash string
	AreaCode     *string
	UpdatedAt    time.Time
	FirstLogin   bool    `json:"firstLogin,omitempty"`
	EmployeeId   *string `json:"employeeId,omitempty"`
	AssignedArea *string `json:"assignedArea,omitempty"`
	CreatedByMoh *string `json:"createdByMoh,omitempty"`
}

type MOHUserSummary struct {
	UserId       string    `json:"userId"`
	EmployeeId   string    `json:"employeeId"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PhoneNumber  string    `json:"phoneNumber"`
	AssignedArea string    `json:"assignedArea"`
	FirstLogin   bool      `json:"firstLogin"`
	CreatedAt    time.Time `json:"createdAt"`
}

type Child struct {
	ChildId            string    `json:"childId"`
	RegistrationNumber string    `json:"registrationNumber"`
	FirstName          string    `json:"firstName"`
	LastName           string    `json:"lastName"`
	DateOfBirth        string    `json:"dateOfBirth"`
	Gender             string    `json:"gender"`
	BloodGroup         string    `json:"bloodGroup"`
	BirthWeight        *float64  `json:"birthWeight"`
	BirthHeight        *float64  `json:"birthHeight"`
	HeadCircumference  *float64  `json:"headCircumference"`
	ParentId           *string   `json:"parentId"`
	RegisteredBy       *string   `json:"registeredBy"`
	AreaCode           string    `json:"areaCode"`
	AreaName           string    `json:"areaName"`
	VaccinationStatus  string    `json:"vaccinationStatus,omitempty"`
	CreatedAt          time.Time `json:"createdAt"`
}

type ChildDetail struct {
	Child
	MotherName           string `json:"motherName"`
	MotherNIC            string `json:"motherNic"`
	FatherName           string `json:"fatherName"`
	FatherNIC            string `json:"fatherNic"`
	District             string `json:"district"`
	DsDivision           string `json:"dsDivision"`
	GnDivision           string `json:"gnDivision"`
	Address              string `json:"address"`
	ParentWhatsAppNumber string `json:"parentWhatsAppNumber,omitempty"`
}

type Vaccine struct {
	VaccineId      string `json:"vaccineId"`
	Name           string `json:"name"`
	Manufacturer   string `json:"manufacturer"`
	DosageInfo     string `json:"dosageInfo"`
	RecommendedAge int    `json:"recommendedAge"`
	IntervalDays   int    `json:"intervalDays"`
	Description    string `json:"description"`
	IsActive       bool   `json:"isActive"`
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
	RecordId          string    `json:"recordId"`
	ChildId           string    `json:"childId"`
	RecordedDate      string    `json:"recordedDate"`
	AgeInMonths       *int      `json:"ageInMonths,omitempty"`
	Height            *float64  `json:"height"`
	Weight            *float64  `json:"weight"`
	HeadCircumference *float64  `json:"headCircumference"`
	WeightStatus      string    `json:"weightStatus,omitempty"`
	HeightStatus      string    `json:"heightStatus,omitempty"`
	WeightZScore      *float64  `json:"weightZScore,omitempty"`
	HeightZScore      *float64  `json:"heightZScore,omitempty"`
	RecordedBy        string    `json:"recordedBy"`
	Notes             string    `json:"notes"`
	CreatedAt         time.Time `json:"createdAt"`
}

type GrowthChartPoint struct {
	DateOfVisit string   `json:"dateOfVisit"`
	AgeInMonths int      `json:"ageInMonths"`
	Value       *float64 `json:"value"`
	Status      string   `json:"status,omitempty"`
	ZScore      *float64 `json:"zScore,omitempty"`
	Metric      string   `json:"metric"`
}

type GrowthReferencePoint struct {
	AgeInMonths int     `json:"ageInMonths"`
	SDNeg3      float64 `json:"sdNeg3"`
	SDNeg2      float64 `json:"sdNeg2"`
	SDNeg1      float64 `json:"sdNeg1"`
	Median      float64 `json:"median"`
	SDPos1      float64 `json:"sdPos1"`
	SDPos2      float64 `json:"sdPos2"`
	SDPos3      float64 `json:"sdPos3"`
}

type ChildGrowthCharts struct {
	ChildId          string                 `json:"childId"`
	WeightVsAge      []GrowthChartPoint     `json:"weightVsAge"`
	HeightVsAge      []GrowthChartPoint     `json:"heightVsAge"`
	WeightReference  []GrowthReferencePoint `json:"weightReference"`
	HeightReference  []GrowthReferencePoint `json:"heightReference"`
	HistoryTable     []GrowthRecord         `json:"historyTable"`
	ReferenceVersion string                 `json:"referenceVersion,omitempty"`
}

type Notification struct {
	NotificationId string    `json:"notificationId"`
	RecipientId    string    `json:"recipientId"`
	Type           string    `json:"type"`
	Message        string    `json:"message"`
	RelatedChildId *string   `json:"relatedChildId"`
	SentDate       time.Time `json:"sentDate"`
	IsRead         bool      `json:"isRead"`
}

type Report struct {
	ReportId      string    `json:"reportId"`
	ReportType    string    `json:"reportType"`
	GeneratedBy   string    `json:"generatedBy"`
	GeneratedDate time.Time `json:"generatedDate"`
	StartDate     string    `json:"startDate"`
	EndDate       string    `json:"endDate"`
	DownloadUrl   string    `json:"downloadUrl"`
}

type AuditLog struct {
	LogId      string    `json:"logId"`
	UserId     *string   `json:"userId"`
	UserRole   string    `json:"userRole"`
	UserName   string    `json:"userName"`
	Action     string    `json:"action"`
	EntityType string    `json:"entityType"`
	EntityId   string    `json:"entityId"`
	Details    string    `json:"details"`
	Timestamp  time.Time `json:"timestamp"`
	IpAddress  string    `json:"ipAddress"`
}

type ChildLinkInfo struct {
	ChildID              string
	RegistrationNumber   string
	ParentID             *string
	ParentWhatsAppNumber string
}

type ChildLinkOTP struct {
	ID           string
	ChildID      string
	ParentID     string
	OTPHash      string
	ExpiresAt    time.Time
	AttemptCount int
	MaxAttempts  int
	CreatedAt    time.Time
	ConsumedAt   *time.Time
}

type UserMobileChangeOTP struct {
	ID           string
	UserID       string
	NewPhone     string
	OTPHash      string
	ExpiresAt    time.Time
	AttemptCount int
	MaxAttempts  int
	CreatedAt    time.Time
	ConsumedAt   *time.Time
}

type ChildGrowthWHOObservation struct {
	DateOfVisit  string   `json:"dateOfVisit"`
	AgeMonth     int      `json:"ageMonth"`
	Weight       *float64 `json:"weight,omitempty"`
	Height       *float64 `json:"height,omitempty"`
	WeightStatus string   `json:"weightStatus,omitempty"`
	HeightStatus string   `json:"heightStatus,omitempty"`
	WeightZScore *float64 `json:"weightZScore,omitempty"`
	HeightZScore *float64 `json:"heightZScore,omitempty"`
}

type ChildWHOGrowthPayload struct {
	Version      string                            `json:"version"`
	Metadata     map[string]string                 `json:"metadata,omitempty"`
	ChildID      string                            `json:"childId"`
	Sex          string                            `json:"sex,omitempty"`
	Indicators   map[string][]GrowthReferencePoint `json:"indicators"`
	Observations []ChildGrowthWHOObservation       `json:"observations"`
}

type ClinicSchedule struct {
	ClinicId    string    `json:"clinicId"`
	PhmId       string    `json:"phmId"`
	ClinicDate  string    `json:"clinicDate"`
	ClinicType  string    `json:"clinicType"`
	GnDivision  string    `json:"gnDivision"`
	Location    string    `json:"location"`
	Description string    `json:"description,omitempty"`
	Status      string    `json:"status"` // 'scheduled', 'completed', 'cancelled'
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type DueChild struct {
	ChildId            string  `json:"childId"`
	FirstName          string  `json:"firstName"`
	LastName           string  `json:"lastName"`
	RegistrationNumber string  `json:"registrationNumber"`
	DateOfBirth        string  `json:"dateOfBirth"`
	VaccineName        string  `json:"vaccineName"`
	NextDueDate        string  `json:"nextDueDate"`
	ParentId           *string `json:"parentId,omitempty"`
	ParentName         *string `json:"parentName,omitempty"`
	ParentPhone        *string `json:"parentPhone,omitempty"`
	DoseNumber         *int    `json:"doseNumber,omitempty"`
}

type ClinicChild struct {
	ClinicChildId    string    `json:"clinicChildId"`
	ClinicId         string    `json:"clinicId"`
	ChildId          string    `json:"childId"`
	Attended         bool      `json:"attended"`
	AttendanceStatus string    `json:"attendanceStatus"`
	MissedNotified   bool      `json:"missedNotified"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type ParentDueVaccination struct {
	ClinicId           string `json:"clinicId"`
	ClinicDate         string `json:"clinicDate"`
	ClinicLocation     string `json:"clinicLocation"`
	ChildId            string `json:"childId"`
	ChildName          string `json:"childName"`
	RegistrationNumber string `json:"registrationNumber"`
	VaccineName        string `json:"vaccineName"`
	NextDueDate        string `json:"nextDueDate"`
	ClinicReminder     string `json:"clinicReminder"`
}

type PHMDueVaccination struct {
	ScheduleId          string  `json:"scheduleId"`
	ChildId             string  `json:"childId"`
	ChildName           string  `json:"childName"`
	RegistrationNumber  string  `json:"registrationNumber"`
	VaccineId           string  `json:"vaccineId"`
	VaccineName         string  `json:"vaccineName"`
	DueDate             string  `json:"dueDate"`
	Status              string  `json:"status"`
	ParentId            *string `json:"parentId,omitempty"`
	ParentPhone         *string `json:"parentPhone,omitempty"`
	ReminderSent        bool    `json:"reminderSent"`
	MissedNotified      bool    `json:"missedNotified"`
	DueNotificationText string  `json:"dueNotificationText,omitempty"`
}

type ClinicAttendanceAlert struct {
	ClinicId           string  `json:"clinicId"`
	ChildId            string  `json:"childId"`
	ChildName          string  `json:"childName"`
	RegistrationNumber string  `json:"registrationNumber"`
	ParentId           *string `json:"parentId,omitempty"`
	ParentPhone        *string `json:"parentPhone,omitempty"`
	MissedNotified     bool    `json:"missedNotified,omitempty"`
}
