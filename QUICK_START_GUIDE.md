# Quick Start Guide - Admin & MOH Account Creation

## Initial Setup

### 1. Run Database Migrations
Execute the migration scripts in order:

```bash
# 1. Base schema
psql -U postgres -d ncvms -f scripts/00_schema.sql

# 2. Indexes
psql -U postgres -d ncvms -f scripts/01_indexes.sql

# ... other migrations ...

# 3. NEW: Admin role and MOH OTP support
psql -U postgres -d ncvms -f scripts/08_admin_role_and_moh_otp.sql
```

### 2. Create Initial Admin User
Since admin cannot be created via API, create it directly in the database:

```sql
INSERT INTO users (
    id, 
    email, 
    nic, 
    password_hash, 
    role, 
    name, 
    phone_number, 
    address, 
    language_preference
) VALUES (
    'user-admin-initial',
    'admin@moh.lk',
    '000000000V',
    '$2a$10$YourBcryptHashHere', -- Use bcrypt hash of your password
    'admin',
    'System Administrator',
    '+94711234567',
    'Colombo',
    'en'
);
```

### 3. Start the Application
```bash
go run ./cmd/api/main.go
```

Server will start on configured port (default: 8080)

---

## User Role Hierarchy

```
┌─────────────────────────────────────────────┐
│              SYSTEM ADMIN                   │
│  • Creates MOH accounts with OTP            │
│  • Only 1 admin allowed                     │
│  • Cannot create via public registration   │
└────────────────┬────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────┐
│          MOH (Ministry of Health)           │
│  • Created by Admin using OTP workflow      │
│  • First login required (password change)   │
│  • Can create PHM accounts                  │
│  • Can manage system & reports              │
└────────────────┬────────────────────────────┘
                 │
        ┌────────┴────────┐
        ▼                 ▼
   ┌─────────────┐  ┌──────────────┐
   │    PHM      │  │   PARENT     │
   │  Midwife    │  │  User        │
   ├─────────────┤  ├──────────────┤
   │ • Register  │  │ • Register   │
   │   children  │  │   account    │
   │ • Record    │  │ • Link to    │
   │   vaccines  │  │   children   │
   │ • Send      │  │ • View       │
   │   reminders │  │   records    │
   └─────────────┘  └──────────────┘
```

---

## API Endpoints Reference

### Authentication

#### Register (Parent/PHM)
```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "fullName": "John Doe",
  "nic": "123456789V",
  "email": "john@example.com",
  "phoneNumber": "+94711234567",
  "password": "SecurePass123",
  "confirmPassword": "SecurePass123",
  "role": "parent"  // or "phm" - NOT "admin", NOT "moh"
}


response

{
    "email": "test@moh.lk",
    "firstLogin": true,
    "message": "MOH account created successfully",
    "mohUserId": "user-moh-8ff269d0"
}
```

#### Login
```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "usernameOrEmail": "john@example.com",  // or use NIC
  "password": "SecurePass123"
}

Response:
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": { ... },
  "firstLogin": false
}
```

#### Change Password (After First Login)
```http
POST /api/v1/auth/change-password
Authorization: Bearer <token>
Content-Type: application/json

{
  "oldPassword": "TemporaryPassword",
  "newPassword": "NewSecurePass123",
  "confirmPassword": "NewSecurePass123"
}
```

### Admin Operations

#### Create MOH Account - Simplified Temporary Password (NEW - RECOMMENDED ✨)
```http
POST /api/v1/admin/moh-accounts/create
Authorization: Bearer <admin_token>
Content-Type: application/json

{
  "employeeId": "MOH-2024-001",
  "name": "Dr. Ruwan Silva",
  "nic": "987654321V",
  "email": "rsilva@moh.lk",
  "phoneNumber": "+94771234567",
  "assignedArea": "Colombo District"
}

Response (201 Created):
{
  "message": "MOH account created successfully",
  "mohUserId": "user-moh-xyz789",
  "email": "rsilva@moh.lk",
  "tempPassword": "Xy7@pQ2zKm9#Lx1",
  "maskedDestination": "+94***234567",
  "firstLogin": true
}
```

**Benefits:** Single API call, auto-generated secure password, instant account creation, sent via WhatsApp

---

#### Request OTP for MOH Account Creation (LEGACY - Not Recommended)
```http
POST /api/v1/admin/moh-accounts/request-otp
Authorization: Bearer <admin_token>
Content-Type: application/json

{
  "employeeId": "MOH-2024-001",
  "name": "Dr. Ruwan Silva",
  "nic": "987654321V",
  "email": "rsilva@moh.lk",
  "phoneNumber": "+94771234567",
  "assignedArea": "Colombo District"
}

Response (200 OK):
{
  "otpId": "otp-moh-a1b2c3d4",
  "maskedDestination": "***1234567",
  "expiresInSeconds": 300,
  "message": "OTP sent successfully"
}
```

#### Complete MOH Account Creation (LEGACY - Not Recommended)
```http
POST /api/v1/admin/moh-accounts/complete
Authorization: Bearer <admin_token>
Content-Type: application/json

{
  "otpId": "otp-moh-a1b2c3d4",
  "otpCode": "123456",
  "password": "InitialPassword123",
  "confirmPassword": "InitialPassword123"
}

Response (201 Created):
{
  "message": "MOH account created successfully",
  "mohUserId": "user-moh-xyz789",
  "email": "rsilva@moh.lk",
  "firstLogin": true
}
```

---

## Typical Workflows

### Workflow 1: Admin Creates MOH Account (NEW - RECOMMENDED ✨)

```
1. Admin calls: POST /api/v1/admin/moh-accounts/create
   - Provides MOH employee details in one request
   - System generates secure temporary password
   - System creates MOH user account immediately
   - System sends temp password via WhatsApp
   - Response includes temp password and masked phone number

2. Admin confirms account creation (can verify via response)

3. MOH employee receives WhatsApp with temporary password
   - Example message: "Your temporary password is: Xy7@pQ2zKm9#Lx1"

4. MOH employee logs in with email/NIC and temporary password
   - POST /api/v1/auth/login
   - Receives JWT token with firstLogin: true

5. MOH employee must call: POST /api/v1/auth/change-password (REQUIRED)
   - Old password: Xy7@pQ2zKm9#Lx1 (the temporary password)
   - New password: User's own secure password
   - firstLogin flag is set to false after successful change
   - Account fully activated ✅

Advantages:
✅ Single API call (faster)
✅ No manual OTP entry needed
✅ More secure (12+ char password vs 6-digit OTP)
✅ Account created immediately
✅ Better user experience
```

### Workflow 2: Admin Creates MOH Account (LEGACY - OTP-based)

```
1. Admin calls: POST /api/v1/admin/moh-accounts/request-otp
   - Provides MOH employee details
   - System generates 6-digit OTP
   - System sends OTP via WhatsApp
   - System returns otpId and masked phone number

2. OTP received by MOH employee via WhatsApp (valid for 5 minutes)

3. Admin calls: POST /api/v1/admin/moh-accounts/complete
   - Admin provides otpId, OTP code, and password
   - System verifies OTP
   - System creates MOH account
   - MOH user set to firstLogin = true

4. MOH employee logs in with email/NIC and password

5. MOH employee must call: POST /api/v1/auth/change-password
   - Required to complete first login
   - firstLogin flag is set to false
   - Account fully activated

Note: This workflow is maintained for backward compatibility but NEW implementations 
should use Workflow 1 (temporary password) instead.
```

### Workflow 3: MOH Creates PHM Account

```
1. MOH calls: POST /api/v1/users/phm
   - MOH provides PHM employee details
   - System generates temporary password
   - System sends via WhatsApp

2. PHM employee receives temporary password via WhatsApp

3. PHM employee logs in with email/NIC and temporary password

4. PHM employee calls: POST /api/v1/auth/change-password
   - Completes first login
   - Sets own secure password
```

### Workflow 3: Parent Registers

```
1. Parent calls: POST /api/v1/auth/register
   - Parent provides registration details
   - Role set to "parent"
   - Account created immediately (no first login)

2. Parent can immediately log in and access features
```

---

## Error Responses

### 400 Bad Request
```json
{
  "status": 400,
  "code": "BAD_REQUEST",
  "message": "Invalid OTP"
}
```

### 401 Unauthorized
```json
{
  "status": 401,
  "code": "UNAUTHORIZED",
  "message": "Missing or invalid authorization header"
}
```

### 403 Forbidden
```json
{
  "status": 403,
  "code": "FORBIDDEN",
  "message": "Only admin users can create MOH accounts"
}
```

### 409 Conflict
```json
{
  "status": 409,
  "code": "CONFLICT",
  "message": "Email already registered"
}
```

### 422 Validation Error
```json
{
  "status": 422,
  "code": "VALIDATION_ERROR",
  "message": "Validation failed",
  "details": [
    {
      "field": "phoneNumber",
      "message": "Must be a valid phone number"
    }
  ]
}
```

### 429 Too Many Requests
```json
{
  "status": 429,
  "code": "TOO_MANY_REQUESTS",
  "message": "Please wait 45 seconds before requesting another OTP"
}
```

---

## Security Best Practices

### For Admins
- ✅ Keep admin credentials secure
- ✅ Use strong passwords (min 6 chars, prefer 12+)
- ✅ Enable HTTPS in production
- ✅ Regularly rotate admin password
- ✅ Monitor audit logs for suspicious activity
- ✅ Don't share admin credentials

### For MOH Users
- ✅ Change temporary password on first login immediately
- ✅ Don't share OTP codes
- ✅ Use strong passwords
- ✅ Never use same password as other accounts
- ✅ Log out after completing tasks

### For System Operators
- ✅ Backup database regularly
- ✅ Monitor OTP generation rates
- ✅ Keep application updated
- ✅ Review audit logs weekly
- ✅ Test disaster recovery procedures

---

## Configuration Variables

Set these environment variables to customize OTP behavior:

```bash
# OTP Validity (minutes)
export MOBILE_CHANGE_OTP_TTL_MIN=5

# Resend Cooldown (seconds) 
export MOBILE_CHANGE_OTP_COOLDOWN_SEC=60

# Max Verification Attempts
export MOBILE_CHANGE_OTP_MAX_ATTEMPTS=5

# JWT Configuration
export JWT_SECRET="your-secret-key-min-32-chars"
export JWT_EXPIRY_HOURS=24

# Database
export DATABASE_URL="postgresql://user:pass@localhost:5432/ncvms"

# Port
export PORT=8080
```

---

## Troubleshooting

### Issue: "Only one admin user is allowed in the system"
**Cause:** Attempting to insert multiple admin users
**Solution:** Database trigger prevents this. Use existing admin account or delete existing admin first.

### Issue: "OTP not found or expired"
**Cause:** OTP has expired or already been used
**Solution:** Request a new OTP by calling the request endpoint again

### Issue: "OTP attempts exceeded"
**Cause:** Too many failed OTP verification attempts (default: 5)
**Solution:** Request a new OTP from the request endpoint

### Issue: "Admin account cannot be created through registration"
**Cause:** Trying to register with role="admin"
**Solution:** Admin accounts must be created directly in database or via special endpoint only

### Issue: Cannot receive OTP via WhatsApp
**Cause:** Phone number invalid or messaging service not configured
**Solution:** Verify phone number format (+94...) and check TextLK API credentials

---

## Testing with cURL

### Login as Admin
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "usernameOrEmail": "admin@moh.lk",
    "password": "your-admin-password"
  }'

# Save the token from response
export ADMIN_TOKEN="eyJhbGciOiJIUzI1NiIs..."
```

### Create MOH Account (NEW - RECOMMENDED ✨)
```bash
curl -X POST http://localhost:8080/api/v1/admin/moh-accounts/create \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "employeeId": "MOH-2024-001",
    "name": "Dr. Test User",
    "nic": "999999999V",
    "email": "test@moh.lk",
    "phoneNumber": "+94711234567",
    "assignedArea": "Test Area"
  }'

# Response includes tempPassword
# Example response:
# {
#   "message": "MOH account created successfully",
#   "mohUserId": "user-moh-abc123",
#   "email": "test@moh.lk",
#   "tempPassword": "Xy7@pQ2zKm9#Lx1",
#   "maskedDestination": "+94***234567",
#   "firstLogin": true
# }
```

### Request MOH OTP (LEGACY - Not Recommended)
```bash
curl -X POST http://localhost:8080/api/v1/admin/moh-accounts/request-otp \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "employeeId": "MOH-2024-001",
    "name": "Dr. Test User",
    "nic": "999999999V",
    "email": "test@moh.lk",
    "phoneNumber": "+94711234567",
    "assignedArea": "Test Area"
  }'

# Save the otpId from response
export OTP_ID="otp-moh-abc123"
```

### Complete MOH Account (LEGACY - Not Recommended)
```bash
curl -X POST http://localhost:8080/api/v1/admin/moh-accounts/complete \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "otpId": "'$OTP_ID'",
    "otpCode": "123456",
    "password": "NewPassword123",
    "confirmPassword": "NewPassword123"
  }'
```

---

## Database Schema Diagram

```
┌─────────────────────────────────────────┐
│              users                      │
├─────────────────────────────────────────┤
│ id (PK)                                 │
│ email (UNIQUE)                          │
│ nic (UNIQUE)                            │
│ password_hash                           │
│ role: 'parent'|'phm'|'moh'|'admin'     │
│ name                                    │
│ phone_number                            │
│ employee_id (UNIQUE, nullable)          │
│ assigned_area (nullable)                │
│ created_by_moh (FK → users.id)         │
│ first_login (BOOLEAN)                   │
│ created_at, updated_at                  │
└────────────────┬────────────────────────┘
                 │
                 │ admin creates
                 │
                 ▼
┌─────────────────────────────────────────┐
│         moh_account_otps                │
├─────────────────────────────────────────┤
│ id (PK)                                 │
│ admin_id (FK → users.id)               │
│ employee_id                             │
│ email                                   │
│ nic                                     │
│ name                                    │
│ phone_number                            │
│ assigned_area                           │
│ otp_hash (SHA256)                      │
│ attempt_count                           │
│ max_attempts (default: 5)               │
│ consumed_at (nullable)                  │
│ expires_at                              │
│ created_at                              │
└─────────────────────────────────────────┘
```

---

## Support & Documentation

For more information, see:
- `ADMIN_IMPLEMENTATION_SUMMARY.md` - Complete technical details
- `README.md` - General project information
- Postman Collection: `NCVMS_API.postman_collection.json`

---

**Last Updated:** April 2026
**Version:** 1.0 - Initial Admin & MOH OTP Implementation

