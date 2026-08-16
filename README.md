# Backend Prototype — Internal Access Control for Digital Wallet

Backend prototype untuk penelitian Tugas Akhir yang berfokus pada **kontrol akses pengguna internal pada layer aplikasi**, dengan menerapkan **Role-Based Access Control (RBAC)** sebagai kontrol dasar, **Least Privilege (LP)** sebagai pembatas lingkup akses, dan **Just-In-Time (JIT) Access** sebagai mekanisme akses sementara terhadap fitur sensitif.

Repositori ini merupakan artefak implementasi dari penelitian:

**Peneliti:** Muhammad Syukur<br>
**NIM:** 220705058<br>
**Program Studi:** Teknologi Informasi<br>
**Fakultas:** Sains dan Teknologi<br>
**Universitas:** UIN Ar-Raniry Banda Aceh<br>
**Tahun:** 2026

> **Catatan:** Sistem pada repositori ini merupakan **prototipe akademik** yang digunakan untuk merancang, mengimplementasikan, dan memverifikasi mekanisme kontrol akses. Sistem tidak ditujukan sebagai implementasi dompet digital komersial atau sebagai sistem yang siap digunakan pada lingkungan produksi.

---

## Overview

Prototipe merepresentasikan lingkungan operasional backend sistem dompet digital yang melibatkan tiga kategori pengguna:

* **User** — pengguna akhir dan pemilik data.
* **Customer Support (CS)** — pengguna internal yang menangani ticket dan menjadi fokus utama penerapan Least Privilege dan Just-In-Time Access.
* **Administrator** — pengguna internal yang berperan dalam pengelolaan dan pemantauan aktivitas sistem.

Alur operasional utama dibangun menggunakan mekanisme **ticket-based access**, sehingga akses Customer Support terhadap data pengguna tidak diberikan secara global.

Secara konseptual, kontrol akses diterapkan secara berlapis:

```text
Authentication
      │
      ▼
     JWT
      │
      ▼
     RBAC
 Role Validation
      │
      ▼
Least Privilege
Ticket / Assignment Scope
      │
      ▼
Just-In-Time Access
Temporary Sensitive Access
      │
      ▼
 Business Logic
      │
      ▼
   Database
```

RBAC menentukan **siapa** yang dapat memasuki area tertentu, Least Privilege membatasi **ruang lingkup tugas** yang dapat diakses, sedangkan Just-In-Time Access membatasi **fitur sensitif, konteks, dan masa berlaku akses**.

---

# Access Control Model

## 1. Role-Based Access Control (RBAC)

RBAC digunakan sebagai lapisan awal otorisasi.

Endpoint backend dikelompokkan berdasarkan tiga role:

```text
user
cs
admin
```

Setiap request yang telah terautentikasi membawa JWT yang berisi identitas pengguna dan role. Middleware kemudian memverifikasi apakah role tersebut diperbolehkan mengakses endpoint yang diminta.

Contoh pemisahan akses:

```text
/user/*   → role user
/cs/*     → role cs
/admin/*  → role admin
```

Request lintas role ditolak oleh backend dengan respons `403 Forbidden`.

RBAC pada penelitian ini **bukan satu-satunya kontrol akses**. Role CS hanya memberikan akses awal ke area kerja Customer Support, sedangkan akses terhadap ticket dan fitur sensitif tetap membutuhkan validasi tambahan.

---

## 2. Least Privilege

Least Privilege diterapkan untuk membatasi ruang lingkup akses Customer Support agar tetap sesuai dengan kebutuhan penanganan ticket.

Setelah berhasil login, CS tidak mendapatkan akses global terhadap seluruh pengguna atau seluruh data pada sistem.

CS hanya dapat:

* melihat antrean ticket yang tersedia;
* melakukan claim terhadap ticket;
* melihat ticket yang menjadi assignment-nya;
* berinteraksi dengan pengguna dalam konteks ticket tersebut;
* mengajukan akses JIT ketika membutuhkan fitur sensitif.

Backend melakukan validasi terhadap hubungan antara CS yang sedang login dan ticket yang diakses.

Validasi utama meliputi:

```text
ticket_id exists
        │
        ▼
assigned_cs_id == authenticated CS
        │
        ▼
ticket context valid
        │
        ▼
request allowed
```

Dengan mekanisme tersebut, memiliki role `cs` saja tidak cukup untuk membuka ticket milik Customer Support lain.

---

## 3. Just-In-Time Access

Just-In-Time Access digunakan untuk mengendalikan fitur yang dikategorikan sensitif agar tidak tersedia secara permanen pada role Customer Support.

Fitur sensitif pada prototipe meliputi:

```text
VIEW_KYC
RESET_PASSWORD
CHANGE_EMAIL
RESET_PIN
UNBLOCK_ACCOUNT
```

Sebelum menjalankan salah satu fitur tersebut, CS harus memperoleh **JIT session** terlebih dahulu.

### JIT Request Validation

Backend melakukan validasi berikut sebelum session diterbitkan:

1. `ticket_id` harus tersedia pada database.
2. Ticket harus ditugaskan kepada CS yang mengajukan request.
3. Ticket harus berada pada status `IN_PROGRESS`.
4. Feature yang diminta harus termasuk fitur yang dikendalikan melalui JIT.
5. Session aktif sebelumnya untuk kombinasi CS, ticket, dan feature yang sama dinonaktifkan sebelum session baru dibuat.

Jika seluruh kondisi terpenuhi, backend membuat session sementara pada tabel:

```text
jit_sessions
```

Session mengikat:

```text
CS
Ticket
Feature
Active Status
Expiration Time
```

Pada prototipe ini:

```text
JIT lifetime : 15 minutes
Usage        : Single-use
```

Durasi 15 menit merupakan parameter prototipe untuk memverifikasi mekanisme akses sementara dan **bukan standar keamanan baku untuk sistem produksi**.

Session JIT dinonaktifkan apabila:

* fitur sensitif berhasil digunakan;
* session melewati `expired_at`;
* ticket ditutup;
* CS melakukan logout.

Dengan demikian, akses sensitif tidak melekat secara permanen pada role Customer Support.

---

# Ticket Lifecycle

Ticket menggunakan state berikut:

```text
OPEN
  │
  ▼
CLAIMED
  │
  ▼
IN_PROGRESS
  │
  ▼
RESOLVED
  │
  ▼
CLOSED
```

### OPEN

Ticket baru yang belum ditugaskan kepada Customer Support.

### CLAIMED

Ticket telah diambil dan memiliki hubungan assignment dengan CS tertentu.

### IN_PROGRESS

Ticket sedang aktif ditangani.

Status ini menjadi salah satu persyaratan untuk mengajukan session JIT.

### RESOLVED

Permasalahan telah ditangani.

### CLOSED

Ticket telah ditutup dan tidak lagi menjadi konteks operasional untuk akses sensitif.

Transisi state yang tidak sesuai dengan aturan backend akan ditolak.

---

# Backend Architecture

Backend dibangun menggunakan **Golang** dengan framework **Gin** dan disusun dengan pemisahan lapisan yang mengikuti pola Clean Architecture.

Alur utama request:

```text
Client / API Tester
        │
        ▼
    Gin Router
        │
        ▼
 Authentication
        │
        ▼
 Access Control Layer
 ┌───────────────────────┐
 │ RBAC                  │
 │ Least Privilege       │
 │ Just-In-Time Access   │
 └───────────────────────┘
        │
        ▼
 Usecase / Business Logic
        │
        ▼
 Repository Layer
        │
        ▼
      MySQL
```

Validasi kontrol akses dilakukan pada sisi backend sehingga keputusan menerima atau menolak request tidak bergantung pada tampilan frontend.

---

# Project Structure

```text
.
├── cmd/
│   └── api/
│       └── main.go
│
├── config/
│
├── database/
│
├── internal/
│   ├── domain/
│   ├── repository/
│   │   └── mysql/
│   ├── usecase/
│   ├── delivery/
│   │   ├── http/
│   │   └── websocket/
│   └── middleware/
│
├── pkg/
│
├── frontend/
│
├── docker-compose.yml
├── .env.example
└── README.md
```

### Layer

| Directory                     | Responsibility                                     |
| ----------------------------- | -------------------------------------------------- |
| `cmd/api`                     | Application entry point                            |
| `config`                      | Environment configuration                          |
| `database`                    | Database connection, migration, dan seed           |
| `internal/domain`             | Entity, constants, dan interface                   |
| `internal/repository/mysql`   | Data access menggunakan MySQL/GORM                 |
| `internal/usecase`            | Business rules dan application logic               |
| `internal/delivery/http`      | Gin handler dan routing                            |
| `internal/delivery/websocket` | WebSocket untuk chat dan audit event               |
| `internal/middleware`         | JWT, RBAC, Least Privilege, dan JIT validation     |
| `pkg`                         | Shared utility seperti JWT, password, dan response |

---

# Technology Stack

| Component              | Technology              |
| ---------------------- | ----------------------- |
| Backend                | Golang                  |
| HTTP Framework         | Gin                     |
| Database               | MySQL                   |
| ORM                    | GORM                    |
| Authentication         | JWT                     |
| Realtime Communication | Gorilla WebSocket       |
| Frontend Demo          | Next.js                 |
| API Testing            | Postman                 |
| Containerization       | Docker / Docker Compose |

---

# Getting Started

## Requirements

Untuk menjalankan backend secara lokal:

```text
Go
MySQL
Git
```

Atau gunakan Docker untuk menjalankan stack secara terisolasi.

---

## Environment Configuration

Salin konfigurasi contoh:

```bash
cp .env.example .env
```

Contoh konfigurasi:

```env
APP_PORT=8080

JWT_SECRET=change_me_super_secret

CORS_ALLOWED_ORIGINS=http://localhost:3000,http://127.0.0.1:3000

DB_HOST=127.0.0.1
DB_PORT=3306
DB_USER=root
DB_PASSWORD=
DB_NAME=support_system

DB_ROOT_PASSWORD=root
DB_APP_USER=support_app
DB_APP_PASSWORD=support_app_pass

API_PORT=8080
FRONTEND_PORT=3000

APP_API_BASE_URL=http://localhost:8080
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080
```

> Jangan menggunakan secret, password database, atau credential contoh untuk deployment nyata.

---

# Run Locally

Jalankan backend:

```bash
go run ./cmd/api
```

Pada proses startup, aplikasi akan:

* menginisialisasi koneksi database;
* membuat database apabila belum tersedia;
* menjalankan migration;
* menjalankan data migration yang belum tercatat;
* melakukan seed data default ketika diperlukan;
* menjalankan HTTP API.

---

# Docker

Repositori menyediakan Docker Compose untuk menjalankan komponen:

```text
Frontend
API
MySQL
```

Copy environment:

```bash
cp .env.example .env
cp frontend/.env.example frontend/.env
```

Build dan jalankan:

```bash
docker compose up --build -d
```

Service lokal:

```text
Frontend     http://localhost:3000
API          http://localhost:8080
Health Check http://localhost:8080/health
```

Dalam konfigurasi Docker, aplikasi API berkomunikasi dengan MySQL melalui network internal stack. Konfigurasi deployment tetap harus disesuaikan dengan kebutuhan keamanan lingkungan tempat aplikasi dijalankan.

---

# Deployment Notes

Untuk deployment ke server/VPS, konfigurasi berikut umumnya perlu disesuaikan:

```env
CORS_ALLOWED_ORIGINS=https://app.example.com

APP_API_BASE_URL=https://api.example.com
NEXT_PUBLIC_API_BASE_URL=https://api.example.com

JWT_SECRET=replace-with-strong-secret

DB_ROOT_PASSWORD=replace-root-password
DB_APP_PASSWORD=replace-app-password
```

Kemudian:

```bash
git pull

docker compose up -d --build

docker compose logs --tail=100 api
```

Database disimpan menggunakan Docker volume.

Hindari:

```bash
docker compose down -v
```

apabila data database perlu dipertahankan karena opsi `-v` akan menghapus volume yang digunakan stack.

---

# Demo Accounts

Akun berikut digunakan sebagai data demonstrasi pada lingkungan prototipe:

| Role             | Email              | Password   |
| ---------------- | ------------------ | ---------- |
| Customer Support | `cs01@test.com`    | `cs123`    |
| Customer Support | `cs02@test.com`    | `cs123`    |
| User             | `syukur@gmail.com` | `user123`  |
| Administrator    | `admin@test.com`   | `admin123` |

> Credential di atas hanya digunakan untuk demonstrasi dan pengujian prototipe.

---

# API Response Format

Sebagian besar response API menggunakan struktur:

```json
{
  "success": true,
  "message": "...",
  "data": {}
}
```

---

# API Endpoints

## Authentication

### Login

```http
POST /auth/login
```

Request:

```json
{
  "email": "admin@test.com",
  "password": "admin123"
}
```

Response:

```json
{
  "success": true,
  "message": "login berhasil",
  "data": {
    "token": "...",
    "user": {
      "id": 1,
      "email": "...",
      "role": "admin"
    }
  }
}
```

---

### Logout

```http
POST /auth/logout
```

Middleware:

```text
JWT
```

Untuk role Customer Support, logout juga menonaktifkan session JIT aktif yang masih terkait dengan CS tersebut.

---

# User API

Middleware:

```text
JWT
RBAC(user)
```

### Create Ticket

```http
POST /user/tickets
```

### List User Tickets

```http
GET /user/tickets
```

### Close Ticket

```http
POST /user/tickets/:ticket_id/close
```

### Ticket Messages

```http
GET /user/tickets/:ticket_id/messages?limit=50
```

```http
POST /user/tickets/:ticket_id/messages
```

Request:

```json
{
  "message": "halo"
}
```

---

# Customer Support API

Middleware dasar:

```text
JWT
RBAC(cs)
```

### Available Tickets

```http
GET /cs/tickets/open
```

Menampilkan ticket berstatus `OPEN` yang belum memiliki assignment.

---

### Assigned Tickets

```http
GET /cs/tickets/my
```

Menampilkan ticket aktif yang menjadi assignment CS.

---

### Claim Ticket

```http
POST /cs/tickets/:ticket_id/claim
```

Setelah claim berhasil:

```text
OPEN → CLAIMED
```

Backend juga membatasi jumlah ticket aktif yang dapat ditangani CS sesuai aturan operasional prototipe.

---

## Least Privilege Protected Endpoints

Endpoint berikut membutuhkan validasi assignment ticket.

Middleware tambahan:

```text
LP
```

### Ticket Detail

```http
GET /cs/tickets/:ticket_id
```

### Update Ticket Status

```http
POST /cs/tickets/:ticket_id/status
```

Request:

```json
{
  "status": "IN_PROGRESS"
}
```

### Ticket Messages

```http
GET /cs/tickets/:ticket_id/messages?limit=50
```

```http
POST /cs/tickets/:ticket_id/messages
```

Request:

```json
{
  "message": "kami sedang cek"
}
```

### User Profile in Ticket Context

```http
GET /cs/tickets/:ticket_id/user/profile
```

Data yang diberikan pada akses normal tetap dibatasi sesuai konteks ticket dan kebijakan data exposure prototipe.

---

# JIT Access API

## Request JIT Session

```http
POST /cs/tickets/:ticket_id/jit/request
```

Contoh:

```json
{
  "feature": "CHANGE_EMAIL"
}
```

Session hanya diterbitkan apabila:

```text
ticket exists
        +
assigned to authenticated CS
        +
status == IN_PROGRESS
        +
feature is JIT-controlled
```

---

## Sensitive Endpoints

Endpoint sensitif membutuhkan:

```text
JWT
RBAC(cs)
LP
JIT(feature)
```

### View KYC

```http
POST /cs/tickets/:ticket_id/sensitive/view-kyc
```

Feature:

```text
VIEW_KYC
```

---

### Reset Password

```http
POST /cs/tickets/:ticket_id/sensitive/reset-password
```

Feature:

```text
RESET_PASSWORD
```

Request:

```json
{
  "new_password": "passwordbaru123"
}
```

---

### Change Email

```http
POST /cs/tickets/:ticket_id/sensitive/change-email
```

Feature:

```text
CHANGE_EMAIL
```

Request:

```json
{
  "new_email": "new@example.com"
}
```

---

### Reset PIN

```http
POST /cs/tickets/:ticket_id/sensitive/reset-pin
```

Feature:

```text
RESET_PIN
```

Request:

```json
{
  "new_pin": "1234"
}
```

---

### Unblock Account

```http
POST /cs/tickets/:ticket_id/sensitive/unblock-account
```

Feature:

```text
UNBLOCK_ACCOUNT
```

---

# Administrator API

Middleware:

```text
JWT
RBAC(admin)
```

### Audit Logs

```http
GET /admin/audit-logs?level=HIGH&limit=100
```

Audit log digunakan sebagai komponen pendukung observability untuk membantu penelusuran aktivitas penting pada prototipe.

---

# WebSocket

## Ticket Live Chat

```http
GET /ws/chat/:ticket_id?token=<JWT>
```

Server event:

```json
{
  "event": "ticket_message",
  "payload": {
    "ticket_id": 1,
    "sender_id": 2,
    "sender_role": "cs",
    "message": "...",
    "created_at": "..."
  }
}
```

Validasi participant tetap dilakukan oleh backend:

```text
User → hanya ticket miliknya

CS → hanya ticket yang menjadi assignment-nya
```

---

## Real-Time Audit Event

Administrator dapat menerima event audit melalui:

```http
GET /ws/audit?token=<JWT>
```

Contoh event:

```json
{
  "event": "audit_log",
  "payload": {
    "id": 1,
    "user_id": 2,
    "role": "cs",
    "level": "HIGH",
    "action": "JIT_REQUEST",
    "ticket_id": 1,
    "metadata": {},
    "created_at": "..."
  }
}
```

Endpoint ini dibatasi untuk role Administrator.

---

# Audit Logging

Audit log digunakan sebagai komponen pendukung untuk merekam aktivitas tertentu yang relevan terhadap proses kontrol akses.

Contoh aktivitas:

```text
JIT_REQUEST
VIEW_KYC
RESET_PASSWORD
CHANGE_EMAIL
RESET_PIN
UNBLOCK_ACCOUNT
Access Denial
```

Log disimpan pada:

```text
audit_logs
```

dan dapat digunakan untuk membantu penelusuran proses pengujian dan aktivitas backend.

Audit log pada prototipe ini tidak dimaksudkan sebagai implementasi sistem audit keamanan tingkat produksi atau tamper-resistant logging.

---

# Data Exposure Policy

Akses terhadap data pengguna dibatasi berdasarkan konteks operasional.

Contoh kebijakan yang diterapkan:

* nomor telepon pada tampilan normal dapat dimasking;
* informasi profil yang tersedia pada CS dibatasi sesuai kebutuhan ticket;
* data KYC lengkap dikategorikan sebagai data sensitif;
* akses `VIEW_KYC` membutuhkan session JIT yang valid.

Contoh masking:

```text
0812****123
```

Pendekatan ini digunakan untuk merepresentasikan penerapan Least Privilege terhadap data sensitif pada layer aplikasi.

---

# Security Scope

Prototipe dirancang untuk memverifikasi beberapa skenario pembatasan akses internal.

| Scenario                                            | Control                 |
| --------------------------------------------------- | ----------------------- |
| User mengakses area internal                        | RBAC                    |
| CS mengakses area Administrator                     | RBAC                    |
| CS membuka ticket milik CS lain                     | Least Privilege         |
| CS mengakses data tanpa konteks assignment          | Least Privilege         |
| CS menjalankan fitur sensitif tanpa session         | JIT                     |
| Session digunakan untuk feature berbeda             | JIT feature binding     |
| Session digunakan setelah expired                   | JIT expiration          |
| Session digunakan kembali setelah eksekusi          | Single-use JIT          |
| Fitur dijalankan setelah konteks ticket tidak valid | Ticket + JIT validation |

Kontrol tersebut digunakan untuk **membatasi dan memverifikasi skenario akses pada lingkungan prototipe**.

Repositori ini tidak dimaksudkan untuk membuktikan bahwa sistem telah:

* aman secara absolut;
* memenuhi seluruh regulasi finansial;
* siap untuk production deployment;
* tahan terhadap seluruh bentuk serangan;
* atau setara dengan sistem keamanan industri finansial.

---

# Testing Scope

Evaluasi penelitian berfokus pada **backend enforcement** menggunakan pendekatan scenario-based dan black-box testing.

Validasi terutama diamati melalui:

```text
HTTP response
Access decision
Database state
JIT session state
Audit / terminal log
```

Pengujian difokuskan untuk memverifikasi apakah implementasi berjalan sesuai rancangan.

Pengujian penelitian tidak mencakup:

```text
Performance benchmark
Load testing
Penetration testing komprehensif
Infrastructure security assessment
Production readiness assessment
```

---

# Research Context

Implementasi dalam repositori ini menempatkan:

```text
RBAC
↓
sebagai batas awal berdasarkan role

Least Privilege
↓
sebagai pembatas scope berdasarkan assignment ticket

Just-In-Time Access
↓
sebagai pembatas akses sementara terhadap fitur sensitif
```

Kontribusi implementasi berada pada pemodelan dan verifikasi kombinasi kontrol akses tersebut pada **backend application layer** menggunakan konteks operasional Customer Support, ticket, data sensitif, fitur sensitif, dan session JIT.

---

# Repository

```text
https://github.com/SyukurGit/backend-skripsi
```

---

## Author

**Muhammad Syukur**
Information Technology
Faculty of Science and Technology
UIN Ar-Raniry Banda Aceh
2026
