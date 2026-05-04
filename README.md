# Support Backend (Clean Architecture)

Backend ini adalah sistem ticketing + live chat untuk user dan customer support (CS), dengan fokus keamanan:

1) RBAC (Role-Based Access Control)
- Setiap endpoint dilindungi middleware RBAC berdasarkan role `admin`, `cs`, `user`.

2) Least Privilege (LP)
- CS hanya boleh akses ticket yang dia claim.
- WAJIB VALIDASI (di middleware LP dan usecase):
  - `assigned_cs_id` harus sama dengan CS yang login
  - status ticket harus `CLAIMED` atau `IN_PROGRESS`

3) Just-In-Time (JIT)
- Untuk fitur sensitif: `RESET_PASSWORD`, `UNBLOCK_ACCOUNT`, `CHANGE_EMAIL`, `RESET_PIN`.
- Flow:
  1. CS claim ticket
  2. CS request JIT (ticket harus `OPEN` dan assigned ke CS)
  3. Sistem buat session JIT expired 15 menit
  4. Middleware JIT cek session ada, aktif, belum expired
  5. Auto revoke jika expired atau ticket `CLOSED`

Catatan: bagian security logic memiliki komentar Bahasa Indonesia sesuai instruksi.

## Struktur Project

Sesuai Clean Architecture:

- `cmd/api/main.go` (entrypoint)
- `config/` (loader env)
- `database/` (connect, migrate, seed)
- `internal/domain/` (entity, constants, interfaces)
- `internal/repository/mysql/` (repo GORM MySQL)
- `internal/usecase/` (business rules)
- `internal/delivery/http/` (Gin handlers + routing)
- `internal/delivery/websocket/` (Gorilla WS: chat + audit)
- `internal/middleware/` (JWT auth, RBAC, LP, JIT)
- `pkg/` (jwt, password, response)

## Konfigurasi & Menjalankan

1) Buat file `.env` (opsional, bisa pakai default):

```env
APP_PORT=8080
JWT_SECRET=change_me_super_secret

DB_HOST=127.0.0.1
DB_PORT=3306
DB_USER=root
DB_PASSWORD=
DB_NAME=support_system
```

2) Jalankan:

```bash
go run ./cmd/api
```

Saat startup, sistem akan:
- `CREATE DATABASE IF NOT EXISTS` untuk `DB_NAME`
- `AutoMigrate` semua tabel
- seed user default jika tabel `users` masih kosong

Seeder default:
- admin: `admin@example.com` / `admin123`
- cs: `cs@example.com` / `cs123`
- user: `user@example.com` / `user123`

## Endpoint API

Semua response dibungkus:

```json
{"success":true,"message":"...","data":{}}
```

### Auth

- POST `/auth/login` (public)
Request:
```json
{"email":"admin@example.com","password":"admin123"}
```
Response:
```json
{"success":true,"message":"login berhasil","data":{"token":"...","user":{"id":1,"email":"...","role":"admin"}}}
```

- POST `/auth/logout` (JWT required)
Middleware: `JWT`
Efek:
- Jika role `cs`, semua session JIT aktif untuk CS tersebut akan direvoke.

### User (role: user)

Middleware group: `JWT` + `RBAC(user)`

- POST `/user/tickets`
Response: ticket baru.

- GET `/user/tickets`

- POST `/user/tickets/:ticket_id/close`

- GET `/user/tickets/:ticket_id/messages?limit=50`

- POST `/user/tickets/:ticket_id/messages`
Request:
```json
{"message":"halo"}
```

### CS (role: cs)

Middleware group: `JWT` + `RBAC(cs)`

- GET `/cs/tickets/open`
List ticket `OPEN` yang belum assigned.

- POST `/cs/tickets/:ticket_id/claim`
Validasi backend:
- COUNT ticket aktif CS (`CLAIMED` atau `IN_PROGRESS`) < 2

Catatan state machine:
- setelah claim sukses, status ticket menjadi `CLAIMED`

Endpoint berikut dilindungi Least Privilege middleware:
Middleware tambahan: `LP` (assigned_cs_id harus sama dan status `CLAIMED`/`IN_PROGRESS`)

- POST `/cs/tickets/:ticket_id/status`
Request:
```json
{"status":"IN_PROGRESS"}
```
Status yang diizinkan: `IN_PROGRESS`, `RESOLVED`, `CLOSED`.

- GET `/cs/tickets/:ticket_id/messages?limit=50`

- POST `/cs/tickets/:ticket_id/messages`
Request:
```json
{"message":"kami sedang cek"}
```

- GET `/cs/tickets/:ticket_id/user/profile`
Audit action: `VIEW_KYC`.

- POST `/cs/tickets/:ticket_id/jit/request`
Request:
```json
{"feature":"RESET_PASSWORD"}
```
Validasi backend:
- ticket harus `CLAIMED` atau `IN_PROGRESS`
- ticket assigned ke CS

Sensitive endpoints (wajib JIT + LP):

Middleware tambahan: `JIT(feature)`

- POST `/cs/tickets/:ticket_id/sensitive/reset-password` (feature `RESET_PASSWORD`)
Request:
```json
{"new_password":"passwordbaru123"}
```

- POST `/cs/tickets/:ticket_id/sensitive/unblock-account` (feature `UNBLOCK_ACCOUNT`)

- POST `/cs/tickets/:ticket_id/sensitive/change-email` (feature `CHANGE_EMAIL`)
Request:
```json
{"new_email":"new@example.com"}
```

- POST `/cs/tickets/:ticket_id/sensitive/reset-pin` (feature `RESET_PIN`)
Request:
```json
{"new_pin":"1234"}
```

### Admin (role: admin)

Middleware group: `JWT` + `RBAC(admin)`

- GET `/admin/audit-logs?level=HIGH&limit=100`
Filter level:
- `HIGH`: `JIT_REQUEST`, `RESET_PASSWORD`, `UNBLOCK_ACCOUNT`
- `MEDIUM`: `VIEW_KYC`, `VIEW_TRANSACTION`

## Flow Utama

1) Login
- Client hit `/auth/login`, simpan JWT.

2) Claim Ticket (CS)
- CS ambil list `/cs/tickets/open`
- CS claim `/cs/tickets/:ticket_id/claim`

State machine ticket (wajib):
- `OPEN -> CLAIMED -> IN_PROGRESS -> RESOLVED -> CLOSED`
- Transisi invalid akan ditolak.

3) JIT Request (CS)
- CS request `/cs/tickets/:ticket_id/jit/request`
- Session aktif 15 menit

4) Eksekusi fitur sensitif
- CS panggil endpoint sensitive sesuai feature
- Middleware JIT memblok jika session tidak ada/expired
- Session auto revoke jika ticket `CLOSED` atau expired

## WebSocket

### Live Chat (user & CS)

- GET `/ws/chat/:ticket_id?token=JWT`
Event server -> client:
```json
{"event":"ticket_message","payload":{"ticket_id":1,"sender_id":2,"sender_role":"cs","message":"...","created_at":"..."}}
```
Client -> server:
- kirim text message biasa (string). Server akan simpan ke DB dan broadcast.

Catatan keamanan:
- VALIDASI: user hanya boleh join ticket miliknya
- VALIDASI: CS hanya boleh join ticket yang assigned ke dia + status `CLAIMED`/`IN_PROGRESS`

### Real-time Audit Log (admin only)

- GET `/ws/audit?token=JWT`
Event:
```json
{"event":"audit_log","payload":{"id":1,"user_id":2,"role":"cs","level":"HIGH","action":"JIT_REQUEST","ticket_id":1,"metadata":{},"created_at":"..."}}
```

Kebijakan pengiriman:
- WS audit hanya mengirim log level `HIGH` dan `MEDIUM`.

## Audit Log System

- Semua aksi penting dipanggil via `AuditUsecase.Log(...)`
- Disimpan ke tabel `audit_logs`
- Dikirim real-time ke admin via WebSocket `/ws/audit`
- Field `audit_logs.level` dipastikan `NOT NULL` dengan default `LOW` dan value dibatasi: `LOW|MEDIUM|HIGH`.

## Data Exposure Policy

- Phone selalu dimasking untuk view default (contoh: `0812****123`).
- KYC default hanya partial (contoh field: `kyc_status`, `is_blocked`, `tier`, `department`).
- Data full (phone + KYC JSON utuh) hanya boleh muncul saat CS punya session JIT aktif untuk `VIEW_KYC`.

Komentar penting di code:
- `// VALIDASI: masking data untuk mencegah kebocoran informasi sensitif`

## Threat Model (ringkas)

- Internal abuse (CS mencoba lihat data user lain): dicegah oleh `LP` + validasi usecase.
- Privilege escalation (CS akses fitur sensitif tanpa JIT): dicegah oleh middleware `JITRequired` + revoke setelah sukses.
- Data leakage via WebSocket: dicegah oleh validasi participant (chat) dan admin-only (audit).

## Panduan untuk Frontend

- Login sekali, simpan JWT, lalu set header:
  - `Authorization: Bearer <token>`
- Untuk chat:
  - Connect ke `/ws/chat/:ticket_id?token=<token>`
  - Render event `ticket_message` sebagai message list
- Admin dashboard:
  - Connect ke `/ws/audit?token=<admin_token>` untuk realtime
  - Gunakan `/admin/audit-logs` untuk initial load + filter
