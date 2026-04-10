# MOH Account Creation - Simplified Temporary Password Workflow

## Overview

This document describes the **NEW simplified workflow** for Admin creating MOH accounts using temporary passwords instead of OTP.

### What Changed?

**OLD Workflow (OTP-based):**
1. Admin requests OTP → System generates & sends OTP
2. Admin waits for OTP code
3. Admin completes account with OTP code & password
4. Takes 2 API calls + manual OTP entry

**NEW Workflow (Temporary Password):**
1. Admin creates account with one API call
2. System generates secure temporary password
3. System creates account immediately
4. System sends temp password via WhatsApp
5. MOH user logs in directly with temp password
6. Takes 1 API call + automatic password send

## Benefits

✅ **Simpler** - Single API call instead of two-step process
✅ **Faster** - No waiting for OTP codes
✅ **More Secure** - Random secure temporary password (12+ chars)
✅ **User-Friendly** - Can login immediately
✅ **Auditable** - Temporary password logged for compliance

## Database Schema

### New Table: `moh_account_temp_passwords`

```sql
CREATE TABLE moh_account_temp_passwords (
    id              TEXT PRIMARY KEY,
    employee_id     TEXT NOT NULL,
    email           TEXT NOT NULL,
    nic             TEXT NOT NULL,
    name            TEXT NOT NULL,
    phone_number    TEXT NOT NULL,
    assigned_area   TEXT NOT NULL,
    admin_id        TEXT NOT NULL REFERENCES users(id),
    temp_password   TEXT NOT NULL,
    used_at         TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### Migration Script

Run the migration to create the new table:

```bash
psql -U postgres -d ncvms -f scripts/11_simplify_moh_creation.sql
```

**Note:** The old `moh_account_otps` table is renamed to `moh_account_otps_deprecated` and kept for historical reference.

## API Endpoint

### Create MOH Account (NEW - Recommended)

**Endpoint:** `POST /api/v1/admin/moh-accounts/create`

**Authentication:** Bearer token (Admin only)

**Request Body:**
```json
{
  "employeeId": "MOH-2024-001",
  "name": "Dr. Ruwan Silva",
  "nic": "987654321V",
  "email": "rsilva@moh.lk",
  "phoneNumber": "+94771234567",
  "assignedArea": "Colombo District"
}
```

**Response (201 Created):**
```json
{
  "message": "MOH account created successfully",
  "mohUserId": "user-moh-a1b2c3d4",
  "email": "rsilva@moh.lk",
  "tempPassword": "Xy7@pQ2zKm9#Lx1",
  "maskedDestination": "+94***234567",
  "firstLogin": true
}
```

**Error Responses:**
```json
// 401 Unauthorized
{
  "status": 401,
  "code": "UNAUTHORIZED",
  "message": "Missing or invalid authorization token"
}

// 403 Forbidden (not admin)
{
  "status": 403,
  "code": "FORBIDDEN",
  "message": "Only admin users can create MOH accounts"
}

// 409 Conflict (email/NIC exists)
{
  "status": 409,
  "code": "CONFLICT",
  "message": "Email already registered"
}

// 500 Error (messaging failed)
{
  "status": 500,
  "code": "ERROR",
  "message": "Failed to send temporary password via WhatsApp"
}
```

## Workflow

```
┌─────────────────────────────────────────────────────────┐
│  1. Admin calls POST /api/v1/admin/moh-accounts/create  │
│     with MOH employee details                           │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│  2. System validates email & NIC are unique             │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│  3. System generates secure temporary password          │
│     - 12 characters (uppercase, lowercase, numbers, !)  │
│     - Example: Xy7@pQ2zKm9#Lx1                          │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│  4. System creates MOH user account                     │
│     - Password: bcrypt hash of temp password            │
│     - firstLogin: true (must change on first login)     │
│     - created_by_moh: admin user ID                     │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│  5. System logs temporary password record               │
│     (expires_at: 24 hours from now)                     │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│  6. System sends temporary password via WhatsApp        │
│     Message: "Your temporary password is: Xy7@pQ..."   │
│     Sent to: +94771234567                              │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│  7. System returns response with temp password          │
│     (visible only in response, not stored in plain)     │
└────────────────────┬────────────────────────────────────┘
                     │
        ┌────────────┴────────────┐
        ▼                         ▼
   Admin sees response      MOH user receives WhatsApp
        │                         │
        ▼                         ▼
   Admin communicates     MOH user logs in with:
   account created        • Email: rsilva@moh.lk
        │                 • Password: Xy7@pQ2zKm9#Lx1
        │                 │
        │                 ▼
        │          ┌─────────────────────────────────┐
        │          │ Must change password on login!  │
        │          │ POST /api/v1/auth/change-pwd   │
        │          └────────────────┬────────────────┘
        │                           │
        │                           ▼
        │          ┌─────────────────────────────────┐
        │          │ firstLogin: false               │
        │          │ Account fully activated         │
        │          └─────────────────────────────────┘
        │
        └────────────────────────────────────────────────► ✅ Complete
```

## Usage Examples

### cURL Examples

**1. Login as Admin**
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "usernameOrEmail": "admin@moh.lk",
    "password": "admin-password"
  }'

# Response includes token
# Save: export ADMIN_TOKEN="eyJhbGc..."
```

**2. Create MOH Account**
```bash
curl -X POST http://localhost:8080/api/v1/admin/moh-accounts/create \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "employeeId": "MOH-2024-001",
    "name": "Dr. Ruwan Silva",
    "nic": "987654321V",
    "email": "rsilva@moh.lk",
    "phoneNumber": "+94771234567",
    "assignedArea": "Colombo District"
  }'

# Response:
# {
#   "message": "MOH account created successfully",
#   "mohUserId": "user-moh-a1b2c3d4",
#   "email": "rsilva@moh.lk",
#   "tempPassword": "Xy7@pQ2zKm9#Lx1",
#   "maskedDestination": "+94***234567",
#   "firstLogin": true
# }
```

**3. MOH User Logs In**
```bash
# MOH receives WhatsApp with temp password
# MOH user calls login endpoint

curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "usernameOrEmail": "rsilva@moh.lk",
    "password": "Xy7@pQ2zKm9#Lx1"
  }'

# Response includes token and firstLogin: true
# Save: export MOH_TOKEN="eyJhbGc..."
```

**4. MOH User Changes Password**
```bash
curl -X POST http://localhost:8080/api/v1/auth/change-password \
  -H "Authorization: Bearer $MOH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "oldPassword": "Xy7@pQ2zKm9#Lx1",
    "newPassword": "MyNewSecurePass123!",
    "confirmPassword": "MyNewSecurePass123!"
  }'

# Response: firstLogin is now false
```

### Postman Collection

1. Import `NCVMS_API.postman_collection.json` into Postman
2. Navigate to: `Admin > Create MOH Account (NEW)`
3. Update variables:
   - `{{admin_token}}` - from login response
   - Employee details in request body
4. Send request
5. Save `tempPassword` from response
6. Navigate to: `Auth > Login` 
7. Use the tempPassword to login as the new MOH user

## Migration from Old OTP Workflow

If you're transitioning from the old OTP workflow:

### Keep Supporting Both (Recommended for now)

The system still supports both endpoints:

```
OLD (OTP-based):
- POST /api/v1/admin/moh-accounts/request-otp
- POST /api/v1/admin/moh-accounts/complete

NEW (Temporary Password):
- POST /api/v1/admin/moh-accounts/create ✨ RECOMMENDED
```

### Retire Old Workflow (Future)

When ready to fully migrate:

1. Update all clients to use `/create` endpoint
2. Mark old endpoints as deprecated
3. After transition period, remove old endpoints
4. Keep historical data in `moh_account_otps_deprecated` table

## Security Features

### Temporary Password Properties

✅ **Length:** 12 characters minimum
✅ **Complexity:** Mix of uppercase, lowercase, numbers, symbols
✅ **Example:** `Xy7@pQ2zKm9#Lx1`, `Km3!xZ1@pQ9vR4`
✅ **TTL:** 24 hours (configurable)
✅ **Storage:** Hashed in database (never stored in plain text)
✅ **Logging:** Audit trail maintained in temp_password table

### First Login Protection

When MOH user logs in:
- `firstLogin` flag is `true`
- They MUST call `POST /auth/change-password` to complete setup
- Can't proceed with other operations until password is changed
- Change password endpoint validates old password matches temp password

### Audit Trail

All MOH account creations logged in `moh_account_temp_passwords` table:
- Who created it (admin_id)
- When it was created
- When it was used (if at all)
- Expiration time

**Query for audit:**
```sql
SELECT 
    id,
    employee_id,
    email,
    admin_id,
    created_at,
    used_at,
    expires_at
FROM moh_account_temp_passwords
WHERE admin_id = 'user-admin-xyz'
ORDER BY created_at DESC;
```

## Configuration

### Application Settings

In `main.go` AdminHandler initialization:

```go
adminHandler := &handlers.AdminHandler{
    // ...existing fields...
    TempPasswordTTL:    24 * time.Hour,      // Valid for 24 hours
    TempPasswordLength: 12,                  // 12 characters
}
```

### Environment Variables

Add to `.env` if needed:

```bash
# Optional - if using external settings
MOH_TEMP_PASSWORD_TTL_HOURS=24
MOH_TEMP_PASSWORD_LENGTH=12
```

## Troubleshooting

### Issue: "Failed to send temporary password via WhatsApp"

**Cause:** WhatsApp messaging service not configured or failed
**Solution:**
- Check TextLK API credentials in config
- Verify phone number format: `+94XXXXXXXXX`
- Check network connectivity to WhatsApp API
- Review messaging service logs

### Issue: Temporary password doesn't work

**Cause:** User hasn't received it yet or it's been too long
**Solution:**
- Verify phone number in account creation
- Check WhatsApp message logs
- Recreate account if necessary (creates new temp password)

### Issue: Cannot change password after first login

**Cause:** Auth handler not set up correctly
**Solution:**
- Verify `POST /auth/change-password` endpoint exists
- Check JWT token is valid
- Verify old password matches temporary password

### Issue: Account not in firstLogin mode

**Cause:** User was created with different method
**Solution:**
- Verify account was created via `POST /api/v1/admin/moh-accounts/create`
- Check database: `SELECT first_login FROM users WHERE id = 'user-id'`
- Set manually if needed: `UPDATE users SET first_login = true WHERE id = 'user-id'`

## Old OTP Workflow (Deprecated)

The old OTP-based workflow is still available but **NOT RECOMMENDED** for new implementations.

### Old Endpoints (Legacy)

```
POST /api/v1/admin/moh-accounts/request-otp
POST /api/v1/admin/moh-accounts/complete
```

### Why We Changed

| Aspect | OTP Workflow | Temp Password |
|--------|-------------|----------------|
| Calls | 2 (request + complete) | 1 |
| Time | Slower (wait for OTP) | Faster (instant) |
| Complexity | Two-step process | Single step |
| User Experience | Requires manual entry | Automatic send |
| Security | 6-digit OTP | 12+ character password |

## FAQ

**Q: Why not stick with OTP?**
A: Temporary passwords are more secure, simpler to implement, and provide better user experience.

**Q: Can users reuse temp passwords?**
A: No, it's hashed and they MUST change on first login.

**Q: What if user loses temp password?**
A: Admin creates new account (new temp password). Old account remains but new one is active.

**Q: Can I customize temp password format?**
A: Yes, edit `generateTempPassword()` function in `admin.go` for different complexity/length.

**Q: Do temp passwords expire?**
A: Yes, default 24 hours. Users should change it immediately after login.

**Q: Can admin see temp password in API response?**
A: Yes, only in the create response. After that, only hashed version is stored.

## Support

For issues or questions:
1. Check troubleshooting section above
2. Review logs: `[moh-creation]` log lines
3. Check database directly: `moh_account_temp_passwords` table
4. Review admin.go handler code

---

**Last Updated:** April 2026
**Version:** 1.0 - Simplified Temporary Password Workflow
**Status:** ✅ Production Ready

