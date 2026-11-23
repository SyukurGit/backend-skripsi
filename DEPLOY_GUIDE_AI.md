# AI Agent Deployment Guide — Support Backend Demo (`backend-skripsi`)

> **ATURAN UTAMA UNTUK AI AGENT:**
> **JANGAN LANGSUNG INSTALL, EDIT `.env` / COMPOSE, BUILD, ATAU START CONTAINER.**
> Jalankan **Tahap 1 (Verifikasi VPS)** terlebih dahulu, laporkan hasilnya ke user, lalu ajukan **Tahap 2 (Daftar Pertanyaan Wajib)** dan tunggu persetujuan user sebelum mengeksekusi perubahan apa pun.

---

## 1. Konteks Arsitektur & Target Infra Clean
- **Sifat Repo:** Public academic / demo prototype (`README.md`). File `.env` demo sengaja disertakan agar mudah dijalankan, dan aplikasi otomatis menjalankan migrasi schema + seed data akun demo (`database/migrate.go`, `database/seed.go` → `SeedIfEmpty`).
- **Stack:** Go (Gin + WebSocket) + MySQL 8.4 via Docker Compose (`docker-compose.backend.yml`).
- **Konfigurasi Compose (`docker-compose.backend.yml`):**
  - Service `mysql` (`mysql:8.4`): `restart: unless-stopped`, bind `127.0.0.1:3306:3306`, volume `mysql_data:/var/lib/mysql`.
  - Service `api` (build `Dockerfile`): `restart: unless-stopped`, bind `127.0.0.1:8080:8080`, `APP_PORT: 8080`.
  - Health endpoint API: `GET /health`
- **Prinsip Infrastruktur Bersih:**
  1. **Bind ke `127.0.0.1` saja:** Port mapping Docker Compose tetap `127.0.0.1:8080:8080` dan `127.0.0.1:3306:3306` (jangan ekspos `0.0.0.0`), serta jangan buka port `8080` atau `3306` di firewall/UFW publik.
  2. **Cloudflare Tunnel Langsung (Tanpa Nginx / HTTP Publik):** Berbeda dari setup lama yang memakai Nginx di port `:80`, untuk setup baru arahkan **Cloudflare Tunnel (`cloudflared`) langsung ke `http://127.0.0.1:8080`**. **Tidak perlu** install Nginx/Apache/Certbot atau membuka port `80`/`443`.
  3. **Tahan VPS Restart:**
     - Kedua container di `docker-compose.backend.yml` sudah memakai `restart: unless-stopped`.
     - Pastikan `docker.service` enabled on boot (`systemctl enable docker`), dan opsional bisa ditambahkan unit `systemd` wrapper jika user menginginkan.

---

## 2. Tahap 1 — Verifikasi Lingkungan VPS (WAJIB DILAKUKAN PERTAMA)
Sebelum melakukan perubahan apa pun, cek dan rangkum kondisi VPS saat ini:
1. **OS, Disk, RAM:** `cat /etc/os-release`, `df -h`, `free -h`.
2. **Docker & Tools:** Cek apakah `docker`, `docker compose` (plugin v2), `curl`, dan `cloudflared` sudah terpasang, serta apakah `docker.service` active & enabled on boot.
3. **Port Target:** Jalankan `ss -tulpn` untuk memastikan port `8080` (API) dan `3306` (MySQL, jika tidak di-expose atau diganti) belum bentrok dengan service lain di host.
4. **Cloudflare Tunnel Existing (KRUSIAL):**
   - Cek apakah `cloudflared` sudah terpasang, sudah login (`cert.pem`), dan sudah punya `config.yml` (di `~/.cloudflared/config.yml` atau `/etc/cloudflared/config.yml`) serta service `cloudflared` yang sedang aktif.
   - **Peringatan Keras:** Jika tunnel sudah ada untuk service lain di VPS ini, agent **WAJIB menambahkan** rule `ingress` baru ke config yang ada (sebelum baris fallback `http_status:404`), **JANGAN PERNAH menimpa atau menghapus** rule hostname yang sudah berjalan.

---

## 3. Tahap 2 — Daftar Pertanyaan Wajib ke User (STOP & TANYA DULU)
Setelah melaporkan hasil Tahap 1, berhenti dan tanyakan ke user:
1. **Domain / Subdomain Endpoint:** Apa nama hostname/subdomain publik untuk API demo ini (atau cukup localhost saja tanpa tunnel)?
2. **Status Cloudflare Tunnel:**
   - Jika pakai domain publik: apakah sudah ada tunnel aktif di VPS ini yang tinggal ditambah rule `ingress`-nya, atau perlu setup/login `cloudflared` baru?
3. **Port Lokal:**
   - Tetap pakai `127.0.0.1:8080` (API) dan `127.0.0.1:3306` (MySQL), atau perlu ubah/hapus expose port `3306` di host (karena `api` sudah terhubung ke `mysql` lewat Docker internal network `support_net`)?
4. **CORS & Environment Demo (`.env`):**
   - Apa domain/origin frontend baru yang perlu dimasukkan ke `CORS_ALLOWED_ORIGINS`?
   - Apakah konfigurasi demo di `.env` langsung dipakai atau mau diganti `JWT_SECRET` / password DB-nya?

---

## 4. Tahap 3 — Eksekusi (Hanya Setelah User Konfirmasi)
1. Sesuaikan `.env` (`CORS_ALLOWED_ORIGINS`, dll.) dan `docker-compose.backend.yml` (jika port host berubah atau port `3306` host dihapus) sesuai jawaban user.
2. Pastikan `systemctl enable --now docker`, lalu jalankan:
   `docker compose -f docker-compose.backend.yml up -d --build`
3. Tambahkan ingress hostname baru ke `cloudflared` mengarah ke `http://127.0.0.1:<API_PORT>` (WebSocket di `/ws/*` otomatis ikut berjalan lewat tunnel), validasi config (`cloudflared tunnel ingress validate`), lalu restart `cloudflared`.
4. **Verifikasi Akhir:**
   - `docker compose -f docker-compose.backend.yml ps` → container `mysql` dan `api` berstatus `healthy`
   - `ss -tulpn | grep 8080` → hanya listen di `127.0.0.1:8080`
   - `curl -s http://127.0.0.1:8080/health` → OK (dan cek log seed/migrasi awal via `docker logs`)
   - Tes akses endpoint HTTPS lewat domain Cloudflare Tunnel baru.
