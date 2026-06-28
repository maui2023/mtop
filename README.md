# Mtop (Monitor System TUI for Proxmox LXC)

Mtop adalah aplikasi pemantauan sistem berasaskan terminal (TUI) yang ringan dan diinspirasikan oleh Btop, direka khas untuk berjalan di dalam kontainer Proxmox LXC mahupun pada perkakasan fizikal (Bare Metal).

Kelebihan utama Mtop ialah kebolehannya membaca had pengehadan **cgroup v2** secara automatik. Jika Mtop dikesan berjalan di dalam LXC, ia hanya akan memaparkan had dan penggunaan sumber (CPU, Memori, Swap) kontainer tersebut sahaja, bukannya keseluruhan sumber hos Proxmox (mengelakkan paparan core CPU hos yang mengelirukan).

## Ciri-Ciri Utama
- **Penapisan CPU Core:** Papar core yang disetkan sahaja pada LXC (cth. jika LXC diset 2 core, hanya 2 core yang dipaparkan walaupun hos mempunyai 64 core).
- **Pengiraan Memori Tepat:** Menapis data RAM dan Swap berdasarkan had cgroup v2 dengan memintas cache yang boleh dibersihkan (seperti `inactive_file`).
- **Penapisan Disk & Rangkaian:** Menunjukkan storan rootfs (`/`) dan rangkaian kontainer sahaja, mengaburkan storan luaran hos atau interface maya hos (`veth*`).
- **Papan Pemuka Moden:** Reka bentuk UI dengan rounded borders dan progress bar unicode licin menggunakan pustaka **Charm Bubble Tea** dan **Lipgloss**.
- **Isihan Proses Dinamik:** Isih proses mengikut `CPU%`, `Memory%`, `PID` atau `Name` dengan menekan kekunci `s`.

## Cara Pemasangan & Pembinaan

### Prasyarat
- Go (Golang) versi 1.18 ke atas.

### Bina Projek (Tempatan)
Jalankan arahan berikut untuk membina binary tempatan:
```bash
make build
```
Ia akan menghasilkan executable binary bernama `mtop`. Jalankan dengan:
```bash
./mtop
```

### Kompilasi Silang (Cross-compilation)
Untuk membina binaan release bagi seni bina sistem AMD64 dan ARM64 (sangat sesuai untuk pelbagai perkakasan homelab):
```bash
make release
```
Binaan executable akan dihasilkan dalam format `mtop-linux-amd64` dan `mtop-linux-arm64`. Anda hanya perlu memindahkan satu fail binary ini ke dalam mana-mana kontainer LXC untuk menggunakannya secara terus (tiada dependensi tambahan diperlukan!).

## Penggunaan Papan Kekunci
- `q` atau `Ctrl+C`: Keluar dari aplikasi.
- `s`: Tukar susunan isihan proses (CPU, Memori, PID, Nama).
- `j` / `k` atau `Panah Bawah` / `Panah Atas`: Skrol senarai proses.
- `r`: Kemas kini statistik secara manual (statistik juga dikemas kini secara automatik setiap 1 saat).
