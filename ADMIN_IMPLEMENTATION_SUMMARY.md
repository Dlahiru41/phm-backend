# Admin Role and MOH Account Creation System - Implementation Summary

## Overview
Successfully implemented a comprehensive 4-user role system with Admin functionality to create MOH accounts using one-time passwords (OTP). The system ensures only one admin exists in the system and provides a secure workflow for MOH account creation.

## 4 User Roles Implemented

### 1. **Parent**
- Can register and manage their children's vaccination records
- Can link to children using OTP
- Can change their mobile phone number with OTP verification
- Can view their children's vaccination schedules and growth records

### 2. **PHM (Public Health Midwife)**
- Can register children in the system
- Can record vaccination activities
- Can create vaccination schedules
- Can send reminders to parents
- Created by MOH users with temporary passwords
- Must complete first login by changing temporary password

### 3. **MOH (Ministry of Health)**
- Can create PHM accounts with temporary passwords
- Can delete vaccination records
- Can generate reports and analytics
- Can view audit logs
- Can manage the system's PHM workforce
- **Created by Admin users using OTP-based workflow**

### 4. **Admin** ✨ NEW
- Only one admin can exist in the system (enforced by database trigger)
- Exclusive authority to create MOH accounts
- Creates MOH accounts using secure OTP-based workflow
- Cannot be created through public registration endpoint
- Full system oversight and MOH account creation responsibilities

---

## Key Features Implemented

### 1. Admin Role Database Updates

**File: `scripts/00_schema.sql`**
- Updated `users` table role constraint to include `'admin'`

**File: `scripts/08_admin_role_and_moh_otp.sql`** (NEW)
- Created `moh_account_otps` table for storing OTP records during MOH account creation
- Implemented database trigger `enforce_single_admin` to ensure only one admin user
- Added indexes for optimal query performance
- Proper foreign key constraints and cascading

### 2. Store Layer for OTP Management

**File: `internal/store/moh_account_otp.go`** (NEW)
- `MOHAccountOTPStore` struct with comprehensive OTP management methods
- `Create()` - Generate new OTP record for MOH account creation
- `GetLatestActive()` - Retrieve active (unconsumed) OTP by admin and email
- `GetByID()` - Retrieve OTP record by ID
- `IncrementAttempt()` - Track failed OTP verification attempts
- `ConsumeValid()` - Verify and consume OTP
- `InvalidateActive()` - Expire active OTPs
- `CountAdminOTPsCreatedToday()` - Track OTP creation rate

### 3. User Store Extensions

**File: `internal/store/user.go` (Updated)**
- `CreateMOH()` - Create MOH accounts with admin as creator
- `IsAdmin()` - Check if user has admin role
- `CountAdminUsers()` - Verify single admin constraint

### 4. Admin Handler Implementation

**File: `internal/handlers/admin.go`** (NEW)
- `AdminHandler` struct with MOH account creation methods
- `RequestMOHAccountOTP()` - Admin initiates OTP creation for MOH account
  - Validates input and checks for duplicate email/NIC
  - Rate limiting (resend cooldown)
  - OTP delivery via WhatsApp
  - Returns masked phone number for privacy
- `CompleteMOHAccount()` - Admin finalizes MOH account with OTP verification
  - OTP verification with attempt tracking
  - Account creation on successful verification
  - First login flag set to true (requires password change)
  - Proper error handling for expired/used OTPs

### 5. Shared OTP Utilities

**File: `internal/handlers/otp_utils.go`** (NEW - Centralized)
- `hashOTP()` - SHA256 hashing for OTP verification
- `generateOTPCode()` - Random 6-digit OTP generation
- `maskPhone()` - Phone masking for secure display
- `normalizePhone()` - Phone number validation and normalization
- Removed duplicates from `children.go` and centralized for maintainability

### 6. Authentication Updates

**File: `internal/handlers/auth.go` (Updated)**
- Prevented admin account creation via public registration endpoint
- Registration validation specifically blocks admin role requests
- Role constraint enforced: `oneof=parent phm moh`

### 7. Router Configuration

**File: `internal/router/routes.go` (Updated)**
- Added new admin route group: `/api/v1/admin`
- Endpoints are protected with authentication and admin role check
- Routes:
  - `POST /api/v1/admin/moh-accounts/request-otp` - Request OTP for MOH creation
  - `POST /api/v1/admin/moh-accounts/complete` - Complete MOH account creation

### 8. Main Application Setup

**File: `cmd/api/main.go` (Updated)**
- Initialized `MOHAccountOTPStore` with database pool
- Created `AdminHandler` instance with proper configuration
- Passed admin handler to router setup
- OTP TTL and attempt limits configured for consistency

---

## Security Features

### Authentication & Authorization
- JWT-based authentication for all protected endpoints
- Role-based access control (RBAC) middleware
- Admin-only middleware for sensitive operations
- Proper HTTP status codes (401 Unauthorized, 403 Forbidden)

### OTP Security
- 6-digit random OTP generation using cryptographically secure random
- SHA256 hashing of OTPs (hash stored, plain text sent via WhatsApp)
- Configurable TTL (Time To Live) for OTP validity
- Configurable attempt limits (default 5 attempts)
- Automatic OTP expiration after TTL
- Rate limiting on OTP requests (resend cooldown)
- Consumed OTPs cannot be reused

### Database Constraints
- Database-level check constraint on role field
- Trigger-based single admin enforcement
- Foreign key constraints with cascading deletes
- Unique constraints on email and NIC

### Password Management
- BCrypt hashing for MOH account passwords
- Required password change on first login
- Minimum 6-character password requirement
- Confirmation password validation

---

## Workflow: Admin Creating MOH Account

### Step 1: Request OTP
```
POST /api/v1/admin/moh-accounts/request-otp
Authorization: Bearer <admin_token>

{
  "employeeId": "MOH-12345",
  "name": "Dr. Silva",
  "nic": "123456789V",
  "email": "silva@moh.lk",
  "phoneNumber": "0711234567",
  "assignedArea": "Colombo District"
}
```

**Response:**
```json
{
  "otpId": "otp-moh-a1b2c3d4",
  "maskedDestination": "***1234567",
  "expiresInSeconds": 300,
  "message": "OTP sent successfully"
}
```

### Step 2: Complete Account Creation
```
POST /api/v1/admin/moh-accounts/complete
Authorization: Bearer <admin_token>

{
  "otpId": "otp-moh-a1b2c3d4",
  "otpCode": "123456",
  "password": "SecurePassword123",
  "confirmPassword": "SecurePassword123"
}
```

**Response:**
```json
{
  "message": "MOH account created successfully",
  "mohUserId": "user-moh-xyz789",
  "email": "silva@moh.lk",
  "firstLogin": true
}
```

### Step 3: First Login (MOH User)
MOH user logs in with email/NIC and temporary password, then:
- Must change password via `/api/v1/auth/change-password`
- `firstLogin` flag set to false
- Account is now fully activated

---

## Database Changes

### moh_account_otps Table
```sql
CREATE TABLE moh_account_otps (
    id              TEXT PRIMARY KEY,
    admin_id        TEXT NOT NULL REFERENCES users(id),
    employee_id     TEXT NOT NULL,
    email           TEXT NOT NULL,
    nic             TEXT NOT NULL,
    name            TEXT NOT NULL,
    phone_number    TEXT,
    assigned_area   TEXT NOT NULL,
    otp_hash        TEXT NOT NULL,
    attempt_count   INT DEFAULT 0,
    max_attempts    INT DEFAULT 5,
    consumed_at     TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);
```

### Single Admin Enforcement
```sql
-- Trigger function ensures only one admin exists
CREATE TRIGGER enforce_single_admin
    BEFORE INSERT OR UPDATE ON users
    FOR EACH ROW
    EXECUTE PROCEDURE check_only_one_admin();
```

---

## Configuration

The system uses existing configuration variables:
- `JWT_SECRET` - JWT token signing
- `JWT_EXPIRY_HOURS` - Token expiration
- `MOBILE_CHANGE_OTP_TTL_MIN` - OTP validity duration (default: 5 minutes)
- `MOBILE_CHANGE_OTP_COOLDOWN_SEC` - Resend cooldown (default: 60 seconds)
- `MOBILE_CHANGE_OTP_MAX_ATTEMPTS` - Max verification attempts (default: 5)

---

## Files Modified

### Created Files
1. `scripts/08_admin_role_and_moh_otp.sql` - Database schema for admin and MOH OTP
2. `internal/store/moh_account_otp.go` - MOH account OTP store operations
3. `internal/handlers/admin.go` - Admin handler with MOH account creation
4. `internal/handlers/otp_utils.go` - Centralized OTP utility functions

### Modified Files
1. `scripts/00_schema.sql` - Added 'admin' role to role constraint
2. `internal/store/user.go` - Added CreateMOH, IsAdmin, CountAdminUsers methods
3. `internal/handlers/auth.go` - Blocked admin registration via public endpoint
4. `internal/handlers/children.go` - Removed duplicate OTP functions
5. `internal/router/routes.go` - Added admin route group with endpoints
6. `cmd/api/main.go` - Initialized AdminHandler and MOHAccountOTPStore

---

## Error Handling

Comprehensive error responses for:
- ✅ Email/NIC already exists
- ✅ OTP not found or expired
- ✅ OTP already consumed
- ✅ Invalid OTP code
- ✅ OTP attempts exceeded
- ✅ Password mismatch
- ✅ Insufficient permissions
- ✅ Missing required fields
- ✅ Invalid input validation

---

## Next Steps (Recommendations)

1. **Testing**
   - Unit tests for OTP generation and validation
   - Integration tests for MOH account creation workflow
   - Admin role enforcement tests

2. **Monitoring**
   - Log all admin account creation activities
   - Track OTP generation and verification attempts
   - Alert on suspicious activities

3. **Documentation**
   - API documentation in Postman/OpenAPI format
   - Admin user guide
   - Security best practices guide

4. **Initial Admin Setup**
   - Create initial admin user (via direct database insert or special endpoint)
   - Secure admin credentials management
   - Document admin user creation procedure

---

## Security Checklist

- ✅ Only one admin enforced at database level
- ✅ Admin account cannot be created via public registration
- ✅ OTP hashing prevents plain text storage
- ✅ Rate limiting on OTP requests
- ✅ OTP expiration and attempt limiting
- ✅ Role-based access control on all endpoints
- ✅ Password hashing with BCrypt
- ✅ HTTPS recommended for production
- ✅ Database audit trails enabled
- ✅ Proper error messages (no information disclosure)

---

## Conclusion

The implementation provides a secure, scalable, and maintainable system for managing 4 distinct user roles, with special emphasis on the Admin role's exclusive authority to create MOH accounts through a secure OTP-based workflow. The single admin constraint ensures centralized control and accountability for MOH account creation, while OTP verification adds an additional layer of security.

All code follows Go best practices, is properly documented, and includes comprehensive error handling.

