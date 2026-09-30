# Test Cases: Authentication & Registration

This document details the canonical test cases for the Lunar Authentication module covering Login, Registration, Session Management, and Input Constraints.

---

## Positive Scenarios

### AUTH-LOGIN-001
- **Title**: Engineer successfully logs in with valid credentials
- **Classification**: Positive | Priority: P0
- **File**: `frontend/e2e/tests/auth/positive/login.spec.ts`
- **Preconditions**: User account seeded in SQLite repository. Unauthenticated browser session.
- **Steps**:
  1. Navigate to `/login`.
  2. Input valid registered email and password.
  3. Click "Sign in" submit button.
- **Expected Results**:
  - Redirected to `/dashboard`.
  - Session cookie / local token stored.
  - User profile rendered in navigation bar.
- **Evidence**: `docs/qa/evidence/auth/04_login_success_dashboard.png`

### AUTH-LOGIN-003
- **Title**: Engineer can toggle password visibility on login form
- **Classification**: Positive | Priority: P2
- **File**: `frontend/e2e/tests/auth/positive/login.spec.ts`
- **Preconditions**: Unauthenticated browser session on `/login`.
- **Steps**:
  1. Fill password input.
  2. Click password toggle button.
  3. Re-click password toggle button.
- **Expected Results**:
  - Input type switches between `password` and `text`.

### AUTH-SESS-001
- **Title**: Authenticated session state persists across page reload
- **Classification**: Positive | Priority: P0
- **File**: `frontend/e2e/tests/auth/positive/login.spec.ts`
- **Preconditions**: Authenticated user session.
- **Steps**:
  1. Navigate to `/dashboard`.
  2. Execute hard page reload (`page.reload()`).
- **Expected Results**:
  - User remains on `/dashboard` without redirection to `/login`.

### AUTH-NAV-001
- **Title**: Login page contains working navigation link to registration
- **Classification**: Positive | Priority: P1
- **File**: `frontend/e2e/tests/auth/positive/register.spec.ts`
- **Steps**:
  1. Navigate to `/login`.
  2. Click "Create account" link.
- **Expected Results**:
  - URL changes to `/register`.
  - Register form header is visible.

### AUTH-NAV-002
- **Title**: Register page contains working navigation link to login
- **Classification**: Positive | Priority: P1
- **File**: `frontend/e2e/tests/auth/positive/register.spec.ts`
- **Steps**:
  1. Navigate to `/register`.
  2. Click "Sign in" link.
- **Expected Results**:
  - URL changes to `/login`.
  - Login form header is visible.

### AUTH-REG-004
- **Title**: Password strength indicator updates dynamically
- **Classification**: Positive | Priority: P2
- **File**: `frontend/e2e/tests/auth/positive/register.spec.ts`
- **Steps**:
  1. Navigate to `/register`.
  2. Type weak password, then medium password, then strong password.
- **Expected Results**:
  - Password strength bar color and label dynamically transition from Weak to Medium to Strong.

### AUTH-REG-005
- **Title**: Engineer can toggle password visibility on register form
- **Classification**: Positive | Priority: P2
- **File**: `frontend/e2e/tests/auth/positive/register.spec.ts`
- **Steps**:
  1. Input password in register form.
  2. Click visibility eye icon.
- **Expected Results**:
  - Input changes type attribute between `password` and `text`.

### AUTH-REG-009
- **Title**: Engineer successfully registers new account and redirects to dashboard
- **Classification**: Positive | Priority: P0
- **File**: `frontend/e2e/tests/auth/positive/register.spec.ts`
- **Steps**:
  1. Navigate to `/register`.
  2. Enter unique name, valid corporate email, and compliant password.
  3. Submit registration form.
- **Expected Results**:
  - API returns 201 Created with JWT session.
  - Redirected to `/dashboard`.

---

## Negative Scenarios

### AUTH-LOGIN-002
- **Title**: Login with invalid credentials displays error banner and retains inputs
- **Classification**: Negative | Priority: P0
- **File**: `frontend/e2e/tests/auth/negative/login-validation.spec.ts`
- **Steps**:
  1. Navigate to `/login`.
  2. Enter incorrect password for existing user.
  3. Submit login form.
- **Expected Results**:
  - Error banner displays "Invalid credentials" or similar message.
  - Email field retains typed text.
  - Password field is reset.
  - URL remains on `/login`.
- **Evidence**: `docs/qa/evidence/auth/02_login_validation_error.png`

### AUTH-REG-001
- **Title**: Register form shows validation errors on empty submission
- **Classification**: Negative | Priority: P1
- **File**: `frontend/e2e/tests/auth/negative/register-validation.spec.ts`
- **Steps**:
  1. Navigate to `/register`.
  2. Submit form immediately with blank inputs.
- **Expected Results**:
  - Validation messages appear for name, email, and password fields.

### AUTH-REG-002
- **Title**: Register form rejects malformed email formats
- **Classification**: Negative | Priority: P1
- **File**: `frontend/e2e/tests/auth/negative/register-validation.spec.ts`
- **Steps**:
  1. Fill name and password.
  2. Enter invalid email (e.g. `user@invalid`).
  3. Submit form.
- **Expected Results**:
  - Email format validation error is displayed.

### AUTH-REG-003
- **Title**: Register form rejects passwords shorter than 8 characters
- **Classification**: Negative | Priority: P1
- **File**: `frontend/e2e/tests/auth/negative/register-validation.spec.ts`
- **Steps**:
  1. Enter 7-character password.
  2. Trigger blur or submission.
- **Expected Results**:
  - Minimum length requirement error message displayed.

### AUTH-REG-006
- **Title**: Register form rejects mismatched password confirmation
- **Classification**: Negative | Priority: P1
- **File**: `frontend/e2e/tests/auth/negative/register-validation.spec.ts`
- **Steps**:
  1. Enter password `SecurePass123`.
  2. Enter confirmation `DifferentPass456`.
  3. Submit form.
- **Expected Results**:
  - Password mismatch error is displayed.

### AUTH-REG-007
- **Title**: Register form surfaces conflict error when email is already taken
- **Classification**: Negative | Priority: P0
- **File**: `frontend/e2e/tests/auth/negative/register-validation.spec.ts`
- **Steps**:
  1. Register with already existing email.
  2. Submit form.
- **Expected Results**:
  - Backend returns 409 Conflict.
  - Error banner informs user that the account already exists.

### AUTH-REG-008
- **Title**: Register form displays network error banner when backend server is down
- **Classification**: Negative | Priority: P1
- **File**: `frontend/e2e/tests/auth/negative/register-validation.spec.ts`
- **Steps**:
  1. Fill valid registration fields.
  2. Intercept registration API with 500 error or network disconnect.
  3. Submit form.
- **Expected Results**:
  - User-friendly network failure banner surfaces.
