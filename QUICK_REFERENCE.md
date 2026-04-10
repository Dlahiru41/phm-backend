# 🚀 Quick Reference Card

## System Architecture at a Glance

```
┌─────────────────────────────────────────┐
│  4-USER ROLE SYSTEM                     │
├─────────────────────────────────────────┤
│ 1. Parent      - Self-register          │
│ 2. PHM         - Created by MOH         │
│ 3. MOH         - Created by Admin (OTP) │
│ 4. Admin       - Only 1, creates MOH    │
└─────────────────────────────────────────┘
```

---

## Admin MOH Account Creation (2 Steps)

### Step 1️⃣: Request OTP
```bash
curl -X POST http://localhost:8080/api/v1/admin/moh-accounts/request-otp \
  -H "Authorization: Bearer ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "employeeId": "MOH-2024-001",
    "name": "Dr. Silva",
    "nic": "987654321V",
    "email": "silva@moh.lk",
    "phoneNumber": "+94711234567",
    "assignedArea": "Colombo District"
  }'
```
**Response:** `otpId`, masked phone, expires in 300 seconds

### Step 2️⃣: Complete Account
```bash
curl -X POST http://localhost:8080/api/v1/admin/moh-accounts/complete \
  -H "Authorization: Bearer ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "otpId": "otp-moh-a1b2c3d4",
    "otpCode": "123456",
    "password": "SecurePassword123",
    "confirmPassword": "SecurePassword123"
  }'
```
**Response:** `mohUserId`, email, `firstLogin: true`

---

## Database Migrations

```bash
# 1. New MOH OTP table and single admin trigger
psql -U postgres -d ncvms -f scripts/08_admin_role_and_moh_otp.sql

# 2. Initial admin setup guide
psql -U postgres -d ncvms -f scripts/09_initial_admin_setup.sql
```

---

## Create Initial Admin User

```sql
INSERT INTO users (id, email, nic, password_hash, role, name, phone_number, language_preference, first_login)
VALUES (
  'user-admin-' || gen_random_uuid()::text,
  'admin@moh.lk',
  '000000000V',
  '$2a$10$YOUR_BCRYPT_HASH_HERE',  -- Use bcrypt-hashed password
  'admin',
  'System Administrator',
  '+94711234567',
  'en',
  false
);
```

---

## File Quick Reference

### New Files
| File | Purpose | Lines |
|------|---------|-------|
| `08_admin_role_and_moh_otp.sql` | DB schema for OTP & admin | 100 |
| `moh_account_otp.go` | OTP store operations | 180 |
| `admin.go` | Admin request/complete logic | 300 |
| `otp_utils.go` | OTP utilities | 80 |

### Documentation
| File | Purpose | Lines |
|------|---------|-------|
| `ADMIN_IMPLEMENTATION_SUMMARY.md` | Technical details | 332 |
| `QUICK_START_GUIDE.md` | Setup & API reference | 400 |
| `IMPLEMENTATION_COMPLETE.md` | Overview | 300 |
| `IMPLEMENTATION_OVERVIEW.md` | Visual diagrams | 250 |
| `DELIVERABLES.md` | Complete checklist | 350 |

---

## Configuration Variables

```bash
# OTP Settings
MOBILE_CHANGE_OTP_TTL_MIN=5              # 5 minutes validity
MOBILE_CHANGE_OTP_COOLDOWN_SEC=60        # 1 minute resend cooldown
MOBILE_CHANGE_OTP_MAX_ATTEMPTS=5         # 5 failed attempts max

# Security
JWT_SECRET="your-secret-key-min-32-chars"
JWT_EXPIRY_HOURS=24

# Database
DATABASE_URL="postgresql://user:pass@localhost:5432/ncvms"

# Server
PORT=8080
```

---

## API Endpoints Summary

```
┌─ PUBLIC (No Auth) ─────────────────────┐
│ POST /api/v1/auth/login                │
│ POST /api/v1/auth/register             │
└────────────────────────────────────────┘

┌─ AUTHENTICATED (Any Role) ─────────────┐
│ GET  /api/v1/users/me                  │
│ PUT  /api/v1/auth/change-password      │
│ ... (other endpoints)                  │
└────────────────────────────────────────┘

┌─ ADMIN ONLY ───────────────────────────┐
│ POST /api/v1/admin/moh-accounts/       │
│      request-otp                       │
│ POST /api/v1/admin/moh-accounts/       │
│      complete                          │
└────────────────────────────────────────┘

┌─ MOH ONLY ─────────────────────────────┐
│ POST /api/v1/users/phm                 │
│ ... (MOH-specific endpoints)           │
└────────────────────────────────────────┘
```

---

## Security at a Glance

| Feature | Status | Details |
|---------|--------|---------|
| Single Admin | ✅ | Database trigger enforced |
| OTP Hashing | ✅ | SHA256 no plaintext |
| Password | ✅ | BCrypt cost factor 10 |
| Rate Limit | ✅ | 60-second resend cooldown |
| Expiration | ✅ | 5-minute configurable TTL |
| Attempts | ✅ | Max 5 failed attempts |
| RBAC | ✅ | Role-based access control |
| Audit | ✅ | All operations logged |

---

## Workflow at a Glance

```
1. Admin login → GET JWT token
   ↓
2. Admin requests OTP → OTP sent to phone
   ↓
3. Admin verifies OTP → MOH account created
   ↓
4. MOH login → MUST change password
   ↓
5. MOH can now manage system
```

---

## Error Codes

| Code | Meaning | Action |
|------|---------|--------|
| 400 | Bad Request | Check input format |
| 401 | Unauthorized | Log in again |
| 403 | Forbidden | Check role/permissions |
| 409 | Conflict | Email/NIC already exists |
| 422 | Validation Error | Check field requirements |
| 429 | Too Many Requests | Wait before retrying OTP |
| 500 | Server Error | Check logs |

---

## Testing Checklist

- [ ] Admin can log in
- [ ] Admin can request MOH OTP
- [ ] OTP received on phone
- [ ] Admin can complete account with OTP
- [ ] MOH account created successfully
- [ ] MOH can log in
- [ ] MOH must change password
- [ ] Cannot create second admin
- [ ] Admin role cannot be registered

---

## Build Verification

```bash
# Build the project
go build -o app ./cmd/api

# Expected output: No errors, no warnings
# Check exit code: 0
```

---

## Support Documentation

📖 Start here → `QUICK_START_GUIDE.md`
🔧 Technical → `ADMIN_IMPLEMENTATION_SUMMARY.md`
📊 Overview → `IMPLEMENTATION_OVERVIEW.md`
✅ Checklist → `DELIVERABLES.md`

---

## Common Tasks

### Login as Admin
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -d '{"usernameOrEmail":"admin@moh.lk","password":"your-password"}'
```

### Generate Bcrypt Hash
```bash
# Online: https://www.bcryptcalculator.com/
# CLI: echo -n "password" | htpasswd -bnBC 10 "" | tr -d ':\n' | sed 's/\$2y/\$2a/'
```

### Check Admin Exists
```sql
SELECT id, email, role FROM users WHERE role = 'admin';
```

### View OTP Records
```sql
SELECT * FROM moh_account_otps ORDER BY created_at DESC LIMIT 10;
```

---

## Important Notes

⚠️ **Only ONE admin can exist** - Database trigger prevents duplicates
⚠️ **Admin cannot self-register** - Must be created in database
⚠️ **OTP expires in 5 minutes** - Configurable via environment
⚠️ **First login required** - MOH/PHM must change password
⚠️ **HTTPS recommended** - In production use HTTPS
⚠️ **Secure passwords** - Min 6 chars, recommend 12+

---

## Performance Tips

✅ Indexes on `admin_id`, `email`, `expires_at` in moh_account_otps
✅ Database trigger prevents N+1 on admin checks
✅ OTP validity tracked to auto-expire old records
✅ Connection pooling configured in database

---

## Deployment Checklist

- [ ] Database migrations applied
- [ ] Initial admin user created
- [ ] Environment variables set
- [ ] Application built successfully
- [ ] Test admin login works
- [ ] Test OTP workflow
- [ ] Monitor logs for errors
- [ ] Backup database before deployment

---

**Everything is ready to go! 🚀**

Refer to the comprehensive documentation files for detailed information.

