# Project Context - DompetKu Support Security Demo

Dokumen ini dibuat sebagai pegangan utama untuk manusia maupun AI lain agar cepat memahami konteks, tujuan, cakupan, batasan, arsitektur, dan status implementasi project ini secara utuh.

Dokumen ini sengaja lebih luas daripada `README.md` dan `struktur.md`. Jika ingin memahami “mengapa project ini dibuat seperti sekarang”, baca file ini terlebih dahulu.

---

## 1. Identitas Project

- Nama konteks produk: `DompetKu`
- Bentuk project: prototipe aplikasi dompet digital dengan modul customer support internal
- Fokus utama penelitian: implementasi prinsip `Least Privilege (LP)` dan `Just-In-Time Access (JIT)` pada level aplikasi
- Tujuan demo: menunjukkan bahwa akses internal Customer Service terhadap data dan aksi sensitif tidak cukup dikendalikan hanya dengan role, tetapi juga harus dibatasi oleh konteks ticket, status kerja, dan durasi akses

Project ini **bukan** ditujukan sebagai dompet digital production penuh.
Project ini juga **bukan** sekadar aplikasi ticketing biasa.

Project ini adalah:

> prototipe sistem dompet digital yang dipakai untuk mendemonstrasikan pembatasan akses internal berbasis konteks ticket dan pembukaan akses sensitif yang bersifat sementara, terkontrol, dan dapat diaudit.

---

## 2. Masalah yang Ingin Dijawab

Masalah utama yang ingin dijawab project ini adalah:

- role-based access control (`RBAC`) saja tidak selalu cukup untuk membatasi akses internal
- petugas `Customer Service` dapat memiliki akses terlalu luas jika hanya bergantung pada role
- fitur sensitif seperti reset password, reset pin, unblock account, change email, dan akses data KYC tidak seharusnya selalu aktif sepanjang waktu
- seluruh akses sensitif harus dapat dilacak dan dijelaskan ulang saat audit

### Pembeda utama project ini

#### RBAC saja
- hanya menjawab: siapa boleh masuk ke area mana

#### Least Privilege
- menjawab: data mana yang benar-benar boleh diakses dalam konteks kerja tertentu
- di project ini, CS hanya boleh bekerja pada user yang terikat ke ticket yang memang sedang ditangani

#### Just-In-Time Access
- menjawab: kapan fitur sensitif boleh aktif
- di project ini, fitur sensitif default-nya terkunci dan harus diminta terlebih dahulu, hanya aktif sementara, lalu dicabut kembali

---

## 3. Posisi Konteks Produk

Frontend sekarang sengaja diposisikan sebagai:

- **dari sisi user:** aplikasi dompet digital yang realistis secara visual
- **dari sisi CS/admin:** workspace operasional internal untuk mendemonstrasikan LP dan JIT

Artinya:

- saldo dan transaksi user dipakai sebagai konteks produk
- customer support adalah pintu masuk ke skenario keamanan
- inti penelitian ada pada pembatasan akses internal, bukan pada simulasi fitur fintech yang lengkap

### Kenapa ada dummy data?

Dummy data hanya dipakai pada area yang **bukan inti penelitian**, misalnya:

- saldo user
- riwayat transaksi user
- beberapa ringkasan UI produk

Dummy data **tidak boleh** mengganggu atau menggantikan logika utama penelitian, yaitu:

- ticketing
- chat
- profile exposure
- JIT request
- sensitive action
- audit log
- terminal log

---

## 4. Scope Fitur yang Sudah Ada

## 4.1 Auth

- login user/cs/admin
- logout
- session frontend per tab browser
- redirect berbasis role
- 401/session expired handling

## 4.2 User flow

- lihat dashboard dompet digital (dummy context)
- buat ticket bantuan
- buka ticket
- chat dengan CS
- lihat transparansi aktivitas sensitif pada ticket
- close ticket saat status `RESOLVED`

## 4.3 CS flow

- lihat antrean ticket terbuka
- claim ticket
- update status ticket
- chat dengan user
- minta JIT access
- jalankan sensitive action setelah JIT disetujui
- lihat panel LP dan penjelasan backend check
- lihat data pengguna yang awalnya terkunci lalu sebagian dibuka setelah JIT

## 4.4 Admin flow

- dashboard angka ringkas
- logs sesi bantuan per ticket
- realtime audit stream
- terminal log mentah per ticket aktif
- list seluruh akun
- tambah akun `user` dan `cs`

---

## 5. Prinsip Security yang Diimplementasikan

## 5.1 RBAC

Role utama:

- `user`
- `cs`
- `admin`

RBAC membatasi area dasar aplikasi:

- user tidak bisa masuk workspace cs/admin
- cs tidak bisa masuk panel admin
- admin tidak memakai flow ticketing user/cs sebagai pelaku utama

## 5.2 Least Privilege

Project ini tidak berhenti pada role.

LP di project ini berarti:

- CS tidak punya akses global ke seluruh user
- CS hanya boleh mengakses user yang terikat ke ticket yang sedang dia tangani
- ticket harus assigned ke CS yang sedang login
- beberapa operasi juga dibatasi oleh status ticket aktif

Dengan kata lain, akses ditentukan oleh:

- role
- ownership/assignment ticket
- status kerja ticket
- jenis fitur yang diminta

## 5.3 Just-In-Time Access

Fitur sensitif tidak aktif default.

Flow JIT:

1. CS claim ticket
2. CS request feature sensitif tertentu
3. backend cek syarat satu per satu
4. jika valid, backend membuat sesi JIT sementara
5. sesi JIT aktif untuk durasi terbatas
6. setelah aksi sensitif dipakai, sesi di-consume / revoke
7. JIT juga direvoke saat ticket ditutup atau session expired

Fitur sensitif yang saat ini ada:

- `VIEW_KYC`
- `RESET_PASSWORD`
- `UNBLOCK_ACCOUNT`
- `CHANGE_EMAIL`
- `RESET_PIN`

---

## 6. Data Exposure Policy yang Sekarang Berlaku

Ini bagian penting karena sempat dirombak besar.

### Sebelum JIT VIEW_KYC disetujui

CS **tidak langsung** melihat data pengguna.

Yang terjadi:

- panel profile berada pada state `LOCKED`
- nomor telepon tidak dibuka
- saldo tidak dibuka
- data KYC lanjutan tidak dibuka
- UI menjelaskan bahwa akses harus diminta terlebih dahulu

### Setelah JIT VIEW_KYC disetujui

Data **tidak dibuka full mentah**.

Yang dibuka hanya sebagian dengan masking tambahan:

- phone tetap dimasking
- nama dimasking sebagian
- NIK dimasking sebagian
- alamat dimasking sebagian
- beberapa sinyal risiko dan profil akun ditampilkan secara terkontrol

Tujuannya:

- menunjukkan bahwa JIT tidak identik dengan full access
- JIT bekerja di atas LP, bukan menggantikan LP

### Implikasi penting untuk sidang

Kalimat yang aman dipakai:

> Pada penelitian ini, JIT tidak berfungsi sebagai pemberi akses penuh, tetapi sebagai mekanisme pembukaan akses sementara di atas ruang lingkup akses yang tetap dibatasi oleh prinsip Least Privilege.

---

## 7. Transparansi ke User

Project ini tidak hanya fokus pada kontrol dari sisi internal.
User juga diberi transparansi.

Saat event sensitif terjadi, sistem dapat memberi informasi ke user melalui:

- system notification bubble di chat
- panel aktivitas sensitif di halaman detail ticket user

Contoh event yang bisa diketahui user:

- JIT request disetujui
- Customer Service membuka data KYC
- reset password dilakukan
- email diubah
- akun dibuka blokir
- pin direset

Tujuannya:

- menunjukkan bahwa kontrol akses internal dapat diaudit
- menunjukkan bahwa user dapat mengetahui kapan tindakan sensitif terjadi pada sesi bantuannya

---

## 8. Terminal Log dan Audit Log

Project ini punya dua jenis visibilitas backend:

## 8.1 Audit log

Dipakai untuk level aktivitas bisnis dan keamanan yang lebih formal.

Contoh:

- ticket claim
- status update
- JIT request
- JIT denied
- VIEW_KYC
- sensitive action

Audit log dipakai di:

- admin dashboard logs
- admin realtime stream

## 8.2 Terminal log

Dipakai untuk menunjukkan jejak mentah yang lebih dekat ke “cara backend berpikir”.

Fitur utamanya:

- hanya untuk ticket yang sedang diproses
- realtime per ticket
- ada kategori log seperti `JIT`, `STATUS`, `CHAT`, `SENSITIVE`, `FLOW`
- step JIT diurutkan dengan `sequence`, bukan hanya timestamp

Terminal log sangat berguna untuk demo karena bisa menjelaskan:

- request diterima
- validation step 1, 2, 3, dst
- granted / denied
- consume / revoke

---

## 9. Struktur Arsitektur Backend

Project backend memakai pola clean architecture sederhana.

### Folder utama

- `cmd/api/main.go`
  - entry point dan wiring dependency
- `config/`
  - env loader
- `database/`
  - mysql connection, migrate, seed
- `internal/domain/`
  - model, constants, interfaces, DTO
- `internal/repository/mysql/`
  - implementasi repo GORM
- `internal/usecase/`
  - business rules utama
- `internal/delivery/http/`
  - handler HTTP dan route registration
- `internal/delivery/websocket/`
  - hub/handler/publisher websocket
- `internal/middleware/`
  - jwt, rbac, dll
- `pkg/`
  - helper umum seperti password, jwt, response, mask

---

## 10. Struktur Frontend

Frontend menggunakan:

- Next.js App Router
- Tailwind CSS
- komponen UI custom/shadcn-style
- React Query untuk HTTP state
- Zustand untuk auth dan state kecil lain

### Area utama frontend

- `app/`
  - route user, cs, admin, login, landing
- `components/`
  - shell, auth, chat, jit, demo helpers, ui reusable
- `services/`
  - API layer dan query hooks
- `store/`
  - auth, jit, toast
- `utils/`
  - mapper, format, audit helper, env helper

### Prinsip UI yang sekarang dipakai

- user side = konteks produk dompet digital
- cs side = area kerja utama penelitian
- admin side = area pembuktian dan observabilitas
- semua copy diarahkan agar mudah dipakai saat demo sidang

---

## 11. Status Implementasi Backend yang Sudah Ada

## Auth
- login
- logout
- JWT issue
- revoke JIT on logout untuk CS

## User ticketing
- create ticket
- list own tickets
- close resolved ticket
- send/list messages
- list own ticket activity

## CS ticketing
- list open tickets
- list my tickets
- claim ticket
- get ticket detail
- update status dengan state machine
- send/list messages

## CS sensitive flow
- request JIT
- VIEW_KYC gated by JIT
- reset password
- unblock account
- change email
- reset pin

## Admin
- dashboard stats
- sessions list
- session detail
- audit logs
- realtime audit stream
- terminal tickets
- terminal logs per ticket
- user list
- create managed user/cs

---

## 12. Status Implementasi Frontend yang Sudah Ada

## Landing & Login
- landing page dengan positioning dompet digital
- login user dan login staff terpisah
- health check UI
- auth persistence per tab browser

## User
- dashboard dompet digital dummy
- transaksi dan history dummy
- customer support list
- create ticket
- chat ticket
- transparansi aktivitas sensitif

## CS
- dashboard antrian masuk
- my tickets
- chat page
- ticket detail page
- LP explainer panel
- JIT explainer panel
- profile locked/unlocked flow
- CS briefing modal

## Admin
- dashboard angka
- logs per sesi bantuan
- terminal log per ticket aktif
- realtime stream
- users management page

---

## 13. Endpoint Penting yang Sekarang Ada

Daftar ringkas endpoint utama (bukan daftar exhaustif semua variasi query):

### Public
- `GET /health`
- `POST /auth/login`

### Authenticated
- `POST /auth/logout`

### User
- `POST /user/tickets`
- `GET /user/tickets`
- `POST /user/tickets/:ticket_id/close`
- `GET /user/tickets/:ticket_id/messages`
- `POST /user/tickets/:ticket_id/messages`
- `GET /user/tickets/:ticket_id/activity`

### CS
- `GET /cs/tickets/open`
- `GET /cs/tickets/my`
- `POST /cs/tickets/:ticket_id/claim`
- `GET /cs/tickets/:ticket_id`
- `POST /cs/tickets/:ticket_id/status`
- `GET /cs/tickets/:ticket_id/messages`
- `POST /cs/tickets/:ticket_id/messages`
- `GET /cs/tickets/:ticket_id/user/profile`
- `POST /cs/tickets/:ticket_id/jit/request`
- `POST /cs/tickets/:ticket_id/sensitive/reset-password`
- `POST /cs/tickets/:ticket_id/sensitive/unblock-account`
- `POST /cs/tickets/:ticket_id/sensitive/change-email`
- `POST /cs/tickets/:ticket_id/sensitive/reset-pin`

### Admin
- `GET /admin/dashboard`
- `GET /admin/audit-logs`
- `GET /admin/sessions`
- `GET /admin/sessions/:ticket_id`
- `GET /admin/terminal/tickets`
- `GET /admin/terminal/tickets/:ticket_id/logs`
- `GET /admin/users`
- `POST /admin/users`

### WebSocket
- `GET /ws/chat/:ticket_id`
- `GET /ws/audit`
- `GET /ws/admin/terminal/:ticket_id`

---

## 14. Database / Model Penting

### Tabel utama
- `users`
- `user_profiles`
- `tickets`
- `messages`
- `jit_sessions`
- `audit_logs`

### User profile
Saat ini data sensitif banyak direpresentasikan di `user_profiles.kyc_data` (JSON), termasuk:

- kyc status
- block status
- pin hash
- full name
- nik
- birth data
- address
- occupation
- income range
- recent device
- linked bank
- risk score
- dll

### Kenapa JSON?
Karena untuk prototipe ini lebih fleksibel dan cepat dikembangkan, meski secara production real-world normalnya sebagian field sensitif mungkin dipisah atau dinormalisasi lebih lanjut.

---

## 15. Batasan Project

Ini penting agar pembaca tidak salah paham.

Project ini **belum** dimaksudkan sebagai:

- full digital wallet production app
- enterprise PAM lengkap
- sistem approval workflow multi-level penuh
- risk engine adaptif penuh
- SIEM / SOC platform

Yang sudah ada adalah:

- implementasi prinsip LP dan JIT pada level aplikasi
- dengan konteks layanan pelanggan internal
- lengkap dengan audit trail, terminal trace, dan transparansi user yang cukup kuat untuk demo akademik

---

## 16. Risiko Salah Paham yang Harus Dihindari

Saat menjelaskan project ini, hindari framing berikut:

- “ini full fintech app”
- “ini enterprise-grade PAM lengkap”
- “semua hal di sini 100% setara sistem bank sungguhan”

Framing yang lebih aman:

- “ini prototipe sistem dompet digital untuk menguji pembatasan akses internal”
- “kontribusi utama ada pada pembatasan lingkup akses dan durasi akses sensitif”
- “fitur dompet digital dipakai sebagai konteks agar demonstrasi lebih realistis”

---

## 17. Cara Membaca Nilai Penelitian Ini

Nilai project ini **bukan** pada banyaknya fitur.

Nilai utamanya ada pada kombinasi berikut:

1. role saja tidak cukup
2. ticket context dipakai untuk membatasi scope
3. feature sensitif tidak aktif default
4. JIT harus diminta dan divalidasi
5. akses sensitif dibatasi waktu
6. akses sensitif direvoke otomatis
7. user juga dapat transparansi atas tindakan sensitif
8. admin dapat menelusuri seluruh jejak keputusan

Kalau delapan poin itu bisa dipahami, maka esensi project ini sudah terbaca.

---

## 18. Urutan Demo yang Direkomendasikan

Urutan demo paling aman:

1. buka user dashboard sebentar untuk konteks produk
2. masuk ke user ticket untuk menunjukkan jalur bantuan
3. login CS dan buka ticket detail
4. tunjukkan bahwa data profile masih terkunci
5. request `VIEW_KYC`
6. tunjukkan pengecekan JIT
7. tunjukkan data hanya terbuka sebagian
8. tunjukkan user menerima notifikasi sistem
9. jalankan satu aksi sensitif lain
10. buka admin logs
11. buka admin terminal

Kalau waktu singkat, prioritas utama:

- CS ticket detail
- JIT panel
- admin logs
- admin terminal

---

## 19. Hal yang Perlu Dicek Sebelum Demo

- backend direstart setelah perubahan terakhir
- migration dan seed/backfill terbaru sudah jalan
- semua role bisa login normal di tab terpisah
- ada minimal satu ticket aktif untuk demo CS/admin
- flow `VIEW_KYC` sudah dites
- flow satu sensitive action lain sudah dites
- admin logs dan terminal log tampil rapi
- websocket chat dan audit berjalan

---

## 20. File Penting yang Sering Perlu Dibaca

### Backend
- `cmd/api/main.go`
- `internal/delivery/http/routes.go`
- `internal/usecase/cs_usecase.go`
- `internal/usecase/jit_usecase.go`
- `internal/usecase/ticket_usecase.go`
- `internal/usecase/message_usecase.go`
- `internal/usecase/admin_usecase.go`
- `database/migrate.go`
- `database/seed.go`

### Frontend
- `frontend/src/app/cs/tickets/[id]/page.tsx`
- `frontend/src/components/jit/jit-panel.tsx`
- `frontend/src/app/admin/logs/page.tsx`
- `frontend/src/app/admin/terminal/page.tsx`
- `frontend/src/app/user/tickets/[id]/page.tsx`
- `frontend/src/services/queries.ts`
- `frontend/src/utils/audit.ts`
- `frontend/src/utils/map.ts`

---

## 21. Kesimpulan Akhir

Project ini saat ini sudah berada pada posisi:

- layak untuk demo skripsi
- cukup kaya secara visual
- cukup kuat secara alur backend
- punya pembeda yang jelas jika dijelaskan dengan framing yang tepat

Inti yang harus selalu diingat:

> Project ini bukan tentang membuat dompet digital penuh, tetapi tentang memperlihatkan bagaimana akses internal terhadap data dan aksi sensitif dapat dibatasi berdasarkan konteks kerja, waktu, dan jejak audit yang dapat dijelaskan kembali.

Jika pembaca baru hanya mengingat satu hal dari project ini, maka itu seharusnya adalah kalimat di atas.
