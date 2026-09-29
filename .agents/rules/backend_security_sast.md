# Rule: Backend Security, SAST Compliance, and Code Smell Eradication

When developing, refactoring, or reviewing backend code across all services:

1. **SONARQUBE QUALITY GATE STANDARDS**:
   - Maintain Maintainability Grade A: zero code smells.
   - Cognitive complexity must not exceed 15 per function.
   - Cyclomatic complexity must not exceed 10 per function.
   - Zero duplicated string literals (extract repeated tokens, keys, and paths into constants).
   - Zero dead code: eliminate unused variables, unreachable blocks, and abandoned helper functions.

2. **SAST (STATIC APPLICATION SECURITY TESTING) CLEANLINESS**:
   - Zero findings across SAST scanners (e.g. gosec, semgrep, SonarQube, CodeQL).
   - Never ignore SAST warnings with inline bypasses or suppressions unless approved through documented ADR.

3. **COMPREHENSIVE ATTACK MITIGATION**:
   - **SQL / NoSQL / Command Injection**:
     - Parameterized queries exclusively. String concatenation for SQL, NoSQL queries, or shell arguments is strictly prohibited.
     - Never invoke shell interpreters directly with user-influenced arguments.
   - **Broken Object-Level Authorization (BOLA / IDOR)**:
     - Always scope database reads, updates, and deletes by the authenticated user or tenant identity retrieved from verified session state.
     - Never trust client-provided owner IDs without validation.
   - **Server-Side Request Forgery (SSRF)**:
     - Outbound HTTP requests must validate target hostnames against an explicit allowlist.
     - Deny all requests pointing to loopback (`127.0.0.1`), private networks (`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`), or cloud metadata endpoints (`169.254.169.254`).
   - **Path Traversal & Local File Inclusion**:
     - Normalize and sanitize filesystem paths with canonical path resolvers.
     - Ensure resolved paths reside strictly within the allowed root directory.
   - **Denial of Service & ReDoS**:
     - Enforce request body size limits (`http.MaxBytesReader`) on all endpoints accepting payloads.
     - Apply read, write, and idle timeouts on HTTP servers and external HTTP client connections.
     - Avoid vulnerable nested quantifiers in regular expressions.
   - **Race Conditions & Memory Leaks**:
     - Prevent goroutine and thread leaks by binding concurrent operations to `context.Context` cancellation.
     - Close response bodies, database rows, and file handles immediately via deferred calls.
     - Verify race-free concurrency (`go test -race`).
   - **Cryptographic & Secret Safety**:
     - Use cryptographically secure random sources (`crypto/rand`) for security-sensitive tokens.
     - Never hardcode API keys, credentials, or private keys.
     - Never log passwords, authentication headers, or sensitive user data.
