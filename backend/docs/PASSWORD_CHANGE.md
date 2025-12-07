# Password Change Feature

## Overview

Users can now change their own password through the profile page. This is a self-service feature that allows authenticated users to update their password by providing their current password and a new one.

## Backend Implementation

### Endpoint

**Route:** `POST /api/v1/users/password/change`  
**Authentication:** Required (JWT token)  
**Authorization:** Any authenticated user (no admin role required)

### Request Body

```json
{
  "current_password": "string",
  "new_password": "string"
}
```

### Validation

- **Current Password**: Required, must match the user's existing password
- **New Password**: Required, minimum 8 characters

### Response

**Success (200 OK):**
```json
{
  "message": "Password changed successfully"
}
```

**Error Responses:**

- `400 Bad Request`: Invalid request body or new password too short
- `401 Unauthorized`: Not authenticated or current password incorrect
- `404 Not Found`: User not found
- `500 Internal Server Error`: Server error during password update

### Security Features

1. **Current Password Verification**: Users must provide their current password to change it
2. **bcrypt Hashing**: New passwords are hashed using bcrypt with default cost
3. **Audit Logging**: All password change attempts are logged with:
   - Success: `user.password.change`
   - Failed verification: `user.password.change.failed`
   - System errors: `user.password.change.error`

### OAuth Users

Users who only authenticate via OAuth and have not set a local password will receive an error:
```json
{
  "error": "User has no local password set. Please set a password first."
}
```

## Frontend Implementation

### UI Location

The password change feature is accessible from the **Profile Page** (`/profile`).

### UI Components

1. **Security Section**: Displayed only for users with a local password (`has_local_password: true`)
   - Shows last password update date
   - "Change Password" button to open the modal

2. **Password Change Modal**: Contains a form with:
   - Current Password field (PasswordInput)
   - New Password field (PasswordInput, min. 8 characters)
   - Confirm New Password field (PasswordInput, must match new password)
   - Submit button with loading state

### Validation

Frontend validation includes:
- Current password is required
- New password must be at least 8 characters
- Confirm password must match new password

### User Feedback

- **Success**: Green notification "Password changed successfully"
- **Error**: Red notification with the error message from the API
- **Loading State**: Button shows loading spinner during submission
- **Form Reset**: Form is cleared after successful submission or when modal is closed

## Differences from Admin Password Reset

| Feature | User Password Change | Admin Password Reset |
|---------|---------------------|---------------------|
| Endpoint | `/api/v1/users/password/change` | `/admin/users/set-password` |
| Target | Self (authenticated user) | Any user (by admin) |
| Current Password | **Required** | Not required |
| Authorization | Any authenticated user | Admin role required |
| Audit Event | `user.password.change` | `set_user_password` |

## Code Changes

### Backend Files Modified

1. **`backend/internal/handlers/user_handler.go`**
   - Added `ChangePassword()` handler function

2. **`backend/cmd/api/main.go`**
   - Added route: `mux.Handle("/api/v1/users/password/change", authMw.Authenticate(http.HandlerFunc(userHandler.ChangePassword)))`

### Frontend Files Modified

1. **`frontend/src/pages/profile/ProfilePage.tsx`**
   - Added password change modal and form
   - Added `PasswordChangeRequest` interface
   - Added `handlePasswordChange()` function
   - Added Security section in UI (only visible when `has_local_password` is true)

## Testing

### Manual Testing Steps

1. **User with Local Password:**
   - Log in with email/password
   - Navigate to Profile page
   - Click "Change Password"
   - Enter current password
   - Enter new password (min. 8 characters)
   - Confirm new password
   - Submit
   - Verify success notification
   - Log out and log back in with new password

2. **Wrong Current Password:**
   - Follow steps above but enter incorrect current password
   - Verify error notification: "Current password is incorrect"

3. **Password Too Short:**
   - Enter new password with less than 8 characters
   - Verify validation error before submission

4. **Password Mismatch:**
   - Enter different passwords in "New Password" and "Confirm New Password"
   - Verify validation error

5. **OAuth User:**
   - Log in via OAuth (without setting local password)
   - Navigate to Profile page
   - Verify that the Security section is not displayed (as `has_local_password` is false)

## Future Enhancements

- [ ] Password strength indicator
- [ ] Email notification on password change
- [ ] Password history (prevent reusing recent passwords)
- [ ] Optional: Set password for OAuth users who don't have one
- [ ] Password expiration policy
- [ ] Multi-factor authentication requirement for password changes
