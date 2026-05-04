# Struktur & Fitur - Support Backend

Project ini adalah backend ticketing + live chat (HTTP + WebSocket) dengan fokus keamanan:
RBAC, Least Privilege (LP), dan Just-In-Time access (JIT). Arsitektur mengikuti Clean Architecture.

## Ringkasan Fitur

- Auth: login + logout (logout CS revoke semua JIT aktif)
- Ticketing: user buat ticket, CS claim ticket, update status (state machine ketat), user close ticket
- Live chat: user/CS chat per ticket via HTTP dan WebSocket
- JIT access: akses sensitif berbasis ticket, expired 15 menit, auto revoke (expired/close/logout), consume setelah sukses
- Data exposure policy: phone masking + partial KYC default, full hanya saat JIT aktif (VIEW_KYC)
- Audit log: setiap aksi penting dilog, punya level (LOW/MEDIUM/HIGH), realtime WS untuk admin hanya MEDIUM/HIGH

## Struktur Folder (Clean Architecture)

- `cmd/api/main.go`
  - Entry point, wiring dependency, start Gin server
- `config/`
  - Load konfigurasi env (`APP_PORT`, `JWT_SECRET`, `DB_*`)
- `database/`
  - `mysql.go`: koneksi MySQL + bootstrap database
  - `migrate.go`: AutoMigrate + backfill `audit_logs.level` + logging updated rows
  - `seed.go`: seed user default + ticket contoh
- `internal/domain/`
  - Entity/model, constants (role/status/feature/level), interface repo/publisher
- `internal/repository/mysql/`
  - Implementasi repository berbasis GORM
- `internal/usecase/`
  - Business rules (validasi state machine, LP, JIT lifecycle, audit mapping)
- `internal/middleware/`
  - JWT auth, RBAC, LP, JIT required
- `internal/delivery/http/`
  - Handler Gin + route registration
- `internal/delivery/websocket/`
  - WS handler + hub (chat & audit) + publisher
- `pkg/`
  - util: jwt, password, response, mask

## Endpoint Map (HTTP)

Public:
- `GET /health`
- `POST /auth/login`

JWT required:
- `POST /auth/logout`
  - Efek: bila role `cs`, revoke semua JIT session aktif untuk CS tersebut.

User (JWT + RBAC user):
- `POST /user/tickets`
- `GET /user/tickets`
- `POST /user/tickets/:ticket_id/close`
- `GET /user/tickets/:ticket_id/messages?limit=...`
- `POST /user/tickets/:ticket_id/messages`

CS (JWT + RBAC cs):
- `GET /cs/tickets/open` (ticket OPEN yang belum assigned)
- `POST /cs/tickets/:ticket_id/claim`

CS ticket group (JWT + RBAC cs + LP middleware):
- `POST /cs/tickets/:ticket_id/status`
- `GET /cs/tickets/:ticket_id/messages?limit=...`
- `POST /cs/tickets/:ticket_id/messages`
- `GET /cs/tickets/:ticket_id/user/profile`
- `POST /cs/tickets/:ticket_id/jit/request`

CS sensitive (JWT + RBAC cs + LP + JITRequired(feature)):
- `POST /cs/tickets/:ticket_id/sensitive/reset-password` (RESET_PASSWORD)
- `POST /cs/tickets/:ticket_id/sensitive/unblock-account` (UNBLOCK_ACCOUNT)
- `POST /cs/tickets/:ticket_id/sensitive/change-email` (CHANGE_EMAIL)
- `POST /cs/tickets/:ticket_id/sensitive/reset-pin` (RESET_PIN)

Admin (JWT + RBAC admin):
- `GET /admin/audit-logs?level=...&limit=...`

Source of truth routing: `internal/delivery/http/routes.go`

## WebSocket Map

- `GET /ws/chat/:ticket_id?token=JWT`
  - Validasi:
    - user hanya boleh join ticket miliknya
    - CS hanya boleh join ticket assigned ke dia + status CLAIMED/IN_PROGRESS
    - ticket harus aktif untuk chat (OPEN/CLAIMED/IN_PROGRESS)

- `GET /ws/audit?token=JWT`
  - Admin only
  - Broadcast hanya untuk audit level MEDIUM/HIGH

Source: `internal/delivery/websocket/chat_handler.go`, `internal/delivery/websocket/audit_handler.go`

## Struktur Kerja (Flow Utama)

1) Login
- Client -> `POST /auth/login` -> dapat JWT

2) User buat ticket
- `POST /user/tickets` -> ticket status `OPEN`

3) CS claim ticket
- CS lihat `GET /cs/tickets/open`
- CS claim `POST /cs/tickets/:ticket_id/claim`
- Validasi bisnis: CS maksimal 2 ticket aktif (`CLAIMED`/`IN_PROGRESS`)
- Setelah claim sukses: ticket status jadi `CLAIMED`

4) Ticket state machine
- Wajib: `OPEN -> CLAIMED -> IN_PROGRESS -> RESOLVED -> CLOSED`
- Transisi invalid ditolak di usecase
- User hanya boleh close saat status `RESOLVED`

5) Chat
- Bisa via HTTP message endpoints atau WS `/ws/chat/:ticket_id`
- Validasi LP selalu ditegakkan (handler + usecase)

6) JIT untuk fitur sensitif
- CS request `POST /cs/tickets/:ticket_id/jit/request` (ticket harus CLAIMED/IN_PROGRESS + assigned)
- Session expire 15 menit
- Middleware `JITRequired(feature)` enforce sebelum endpoint sensitif
- Auto revoke:
  - saat expired (dibersihkan saat ensure/cek)
  - saat ticket CLOSED
  - saat logout CS
- Consume:
  - setelah aksi sensitif sukses, session direvoke

7) Data Exposure Policy
- Default profile view:
  - phone dimasking
  - KYC ditampilkan partial
- Full data hanya saat JIT aktif feature `VIEW_KYC`, lalu session langsung di-consume.
- Implementasi: `internal/usecase/cs_usecase.go`, `pkg/mask/mask.go`

8) Audit Log
- Semua aksi penting lewat `AuditUsecase.Log`
- Level:
  - HIGH: `JIT_REQUEST`, `RESET_PASSWORD`, `UNBLOCK_ACCOUNT`
  - MEDIUM: `VIEW_KYC`, `VIEW_TRANSACTION`
  - lainnya LOW
- Realtime WS admin hanya MEDIUM/HIGH
- Kolom `audit_logs.level`:
  - dibatasi `LOW|MEDIUM|HIGH` (ENUM)
  - `NOT NULL` + default `LOW`
  - startup migration backfill NULL/empty -> `LOW` dan log jumlah row

## Lapisan Security (Pemetaan)

- JWT: `internal/middleware/jwt.go`
- RBAC: `internal/middleware/rbac.go`
- Least Privilege:
  - Middleware LP ticket CS: `internal/middleware/least_privilege.go`
  - Usecase tetap re-check (defense in depth)
- JIT:
  - Middleware enforce per feature: `internal/middleware/jit.go`
  - Usecase lifecycle: `internal/usecase/jit_usecase.go`

## Model & Storage (Ringkas)

- `internal/domain/models.go`
  - `users`, `user_profiles`, `tickets`, `messages`, `jit_sessions`, `audit_logs`
- Migration: `database/migrate.go`
- Repository MySQL (GORM): `internal/repository/mysql/`

## Cara Jalanin (Dev)

1) Pastikan MySQL hidup dan `.env` terisi (atau pakai default)
2) Run:

```bash
go run ./cmd/api
```

Startup akan connect, migrate, seed jika kosong, lalu listen di `APP_PORT`.
