# API Reference

## Authentication API

All authentication endpoints are handled through Better Auth at `/api/auth/[...all]`

### Base URL
```
Development: http://localhost:3001
Production: https://your-domain.com
```

## Endpoints

### User Registration

#### `POST /api/auth/sign-up`

Creates a new user account with email and password.

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "minimum8chars",
  "name": "John Doe" // optional
}
```

**Response:**
```json
{
  "user": {
    "id": "cuid_example",
    "email": "user@example.com",
    "name": "John Doe",
    "emailVerified": false,
    "createdAt": "2025-01-01T00:00:00Z"
  },
  "session": {
    "id": "session_id",
    "token": "session_token",
    "expiresAt": "2025-01-08T00:00:00Z"
  }
}
```

**Status Codes:**
- `200` - User created successfully
- `400` - Invalid input (email already exists, password too short)
- `500` - Server error

---

### Email Sign In

#### `POST /api/auth/sign-in/email`

Authenticates a user with email and password.

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "yourpassword",
  "callbackURL": "/dashboard" // optional
}
```

**Response:**
```json
{
  "user": {
    "id": "cuid_example",
    "email": "user@example.com",
    "name": "John Doe",
    "emailVerified": true
  },
  "session": {
    "id": "session_id",
    "token": "session_token",
    "expiresAt": "2025-01-08T00:00:00Z"
  }
}
```

**Status Codes:**
- `200` - Sign in successful
- `401` - Invalid credentials
- `403` - Email not verified
- `500` - Server error

---

### Magic Link Sign In

#### `POST /api/auth/sign-in/magic-link`

Sends a magic link to the user's email for passwordless authentication.

**Request Body:**
```json
{
  "email": "user@example.com",
  "callbackURL": "/dashboard" // optional
}
```

**Response:**
```json
{
  "success": true,
  "message": "Magic link sent to email"
}
```

**Status Codes:**
- `200` - Magic link sent
- `400` - Invalid email
- `429` - Rate limit exceeded
- `500` - Server error

---

### Sign Out

#### `POST /api/auth/sign-out`

Terminates the current user session.

**Headers:**
```
Cookie: session-token=your_session_token
```

**Response:**
```json
{
  "success": true
}
```

**Status Codes:**
- `200` - Sign out successful
- `401` - Not authenticated
- `500` - Server error

---

### Get Current Session

#### `GET /api/auth/session`

Retrieves the current authenticated user session.

**Headers:**
```
Cookie: session-token=your_session_token
```

**Response:**
```json
{
  "user": {
    "id": "cuid_example",
    "email": "user@example.com",
    "name": "John Doe",
    "emailVerified": true
  },
  "session": {
    "id": "session_id",
    "expiresAt": "2025-01-08T00:00:00Z"
  }
}
```

**Status Codes:**
- `200` - Session retrieved
- `401` - Not authenticated
- `500` - Server error

---

### Email Verification

#### `GET /api/auth/verify-email`

Verifies a user's email address using a token sent via email.

**Query Parameters:**
- `token` - Verification token from email

**Example:**
```
GET /api/auth/verify-email?token=verification_token_here
```

**Response:**
```json
{
  "success": true,
  "message": "Email verified successfully"
}
```

**Status Codes:**
- `200` - Email verified
- `400` - Invalid or expired token
- `500` - Server error

---

### Forgot Password

#### `POST /api/auth/forgot-password`

Initiates password reset by sending an email with reset link.

**Request Body:**
```json
{
  "email": "user@example.com"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Password reset email sent"
}
```

**Status Codes:**
- `200` - Reset email sent
- `400` - Invalid email
- `429` - Rate limit exceeded
- `500` - Server error

---

### Reset Password

#### `POST /api/auth/reset-password`

Resets user password using token from reset email.

**Request Body:**
```json
{
  "token": "reset_token_here",
  "password": "newpassword123"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Password reset successfully"
}
```

**Status Codes:**
- `200` - Password reset
- `400` - Invalid token or password
- `401` - Token expired
- `500` - Server error

---

## Client SDK Usage

### React Hooks

#### `useSession()`

Gets the current user session.

```typescript
import { useSession } from "@/lib/auth-client";

function Component() {
  const { data: session, isPending } = useSession();
  
  if (isPending) return <div>Loading...</div>;
  if (!session) return <div>Not authenticated</div>;
  
  return <div>Welcome {session.user.email}</div>;
}
```

#### `signIn.email()`

Sign in with email and password.

```typescript
import { signIn } from "@/lib/auth-client";

async function handleSignIn() {
  try {
    await signIn.email({
      email: "user@example.com",
      password: "password123",
      callbackURL: "/dashboard"
    });
  } catch (error) {
    console.error("Sign in failed:", error);
  }
}
```

#### `signIn.magicLink()`

Sign in with magic link.

```typescript
import { signIn } from "@/lib/auth-client";

async function handleMagicLink() {
  try {
    await signIn.magicLink({
      email: "user@example.com",
      callbackURL: "/dashboard"
    });
  } catch (error) {
    console.error("Magic link failed:", error);
  }
}
```

#### `signUp.email()`

Register new user.

```typescript
import { signUp } from "@/lib/auth-client";

async function handleSignUp() {
  try {
    await signUp.email({
      email: "user@example.com",
      password: "password123",
      name: "John Doe",
      callbackURL: "/verify-email"
    });
  } catch (error) {
    console.error("Sign up failed:", error);
  }
}
```

#### `signOut()`

Sign out current user.

```typescript
import { signOut } from "@/lib/auth-client";

async function handleSignOut() {
  await signOut();
  router.push("/login");
}
```

---

## Error Handling

### Error Response Format

```json
{
  "error": {
    "code": "INVALID_CREDENTIALS",
    "message": "Invalid email or password"
  }
}
```

### Common Error Codes

| Code | Description |
|------|-------------|
| `INVALID_CREDENTIALS` | Wrong email or password |
| `EMAIL_NOT_VERIFIED` | Email verification required |
| `USER_NOT_FOUND` | User doesn't exist |
| `EMAIL_ALREADY_EXISTS` | Email already registered |
| `INVALID_TOKEN` | Token invalid or expired |
| `RATE_LIMIT_EXCEEDED` | Too many requests |
| `VALIDATION_ERROR` | Input validation failed |
| `SERVER_ERROR` | Internal server error |

---

## Rate Limiting

### Default Limits

| Endpoint | Limit | Window |
|----------|-------|--------|
| Sign up | 5 requests | 15 minutes |
| Sign in | 10 requests | 15 minutes |
| Magic link | 3 requests | 15 minutes |
| Password reset | 3 requests | 15 minutes |

---

## Security Headers

### Required Headers

```http
Content-Type: application/json
Origin: http://localhost:3001
```

### CORS

Trusted origins must be configured in Better Auth:
- Development: `http://localhost:3000`, `http://localhost:3001`
- Production: Your production domain

---

## WebSocket Events (Future)

For real-time session updates (planned feature):

```javascript
// Future implementation
authClient.on("session.update", (session) => {
  console.log("Session updated:", session);
});

authClient.on("session.expire", () => {
  console.log("Session expired");
});
```

---

## Testing

### Test Credentials

For development only:
```json
{
  "email": "test@example.com",
  "password": "testpass123"
}
```

### cURL Examples

#### Sign Up
```bash
curl -X POST http://localhost:3001/api/auth/sign-up \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"testpass123"}'
```

#### Sign In
```bash
curl -X POST http://localhost:3001/api/auth/sign-in/email \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"testpass123"}'
```

#### Get Session
```bash
curl http://localhost:3001/api/auth/session \
  -H "Cookie: session-token=your_session_token"
```

---

## Migration Guide

### From NextAuth to Better Auth

1. Update imports:
```typescript
// Before
import { signIn } from "next-auth/react";

// After
import { signIn } from "@/lib/auth-client";
```

2. Update API calls:
```typescript
// Before
signIn("credentials", { email, password });

// After
signIn.email({ email, password });
```

3. Update session hooks:
```typescript
// Before
const { data: session } = useSession();

// After (same API)
const { data: session } = useSession();
```