# 🚀 Mtop — Monitor Sistem TUI Premium Khas untuk Proxmox LXC & Bare Metal

[![Go Report Card](https://goreportcard.com/badge/github.com/maui/mtop)](https://goreportcard.com/report/github.com/maui/mtop)
[![Go Version](https://img.shields.io/github/go-mod/go-version/maui/mtop)](https://golang.org)
[![Website](https://img.shields.io/badge/website-mtop.kpst.my-brightgreen)](https://mtop.kpst.my)

**Mtop** adalah aplikasi pemantauan sistem berasaskan terminal (TUI) premium yang ringan dan diinspirasikan oleh keindahan Btop. Dibina sepenuhnya dalam **Go**, Mtop menyelesaikan masalah pemantauan sumber di dalam **Proxmox LXC** dan mesin fizikal (**Bare Metal**) dengan membawakan visualisasi sumber yang tepat, bersih, dan bebas kekeliruan!

🌐 **Layari laman web rasmi kami:** [https://mtop.kpst.my](https://mtop.kpst.my)

---

![Mtop Dashboard](images/mtop-proxmox-lxc.png)

---

## 🔥 Mengapa Mtop Berbeza?

Apabila anda menjalankan pemantau biasa (seperti `htop` atau `btop`) di dalam kontainer LXC, ia sering kali memaparkan maklumat keseluruhan hos yang mengelirukan (seperti 64 core CPU hos atau 256GB RAM hos, sedangkan kontainer anda hanya diperuntukkan 2 core dan 4GB RAM). 

**Mtop menyelesaikan masalah ini sepenuhnya!** Mtop berintegrasi secara pintar dengan **cgroup v2** di peringkat kontainer untuk menapis dan mengehadkan statistik sistem supaya sepadan dengan had peruntukan LXC anda secara automatik.

### ✨ Ciri-Ciri Utama:
- **🎯 Pengesanan Core Tepat:** Hanya tunjukkan CPU core yang disetkan oleh Proxmox untuk kontainer tersebut (cth: papar 2 core sahaja jika had LXC adalah 2, walaupun CPU hos mempunyai 32/64 core).
- **🧠 Pengiraan RAM Lebih Bersih:** Mengira RAM dan Swap sebenar menggunakan had cgroup v2, serta menapis keluar cache memori yang boleh dituntut semula (`inactive_file`) agar statistik memori anda setepat arahan `free -m`.
- **💾 Penapisan Storan Pintar:** Menyembunyikan storan ZFS pool atau LVM hos yang tidak berkaitan, hanya memaparkan storan rootfs (`/`) kontainer dan mount point aktif sahaja.
- **🔌 Rangkaian Terfokus:** Menapis keluar interface maya seperti `veth*` atau `lo` yang berserakan, hanya memaparkan interface rangkaian kontainer yang aktif dan betul (cth: `eth0`).
- **🎨 Visual TUI Premium:** Menggunakan framework **Charm Bubble Tea** & **Lipgloss** untuk rekaan papan pemuka yang moden, lengkap dengan rounded borders dan progress bar unicode bersudut licin.
- **⚡ Pengurusan Proses Lancar:** Senarai proses dinamik yang boleh diskrol (`j`/`k`) dan diisih mengikut `CPU%`, `Memory%`, `PID` atau `Name` dengan menekan kekunci `s`.

---

## 🛠️ Cara Pemasangan & Pembinaan

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

---

## ⌨️ Penggunaan Papan Kekunci
- `q` atau `Ctrl+C`: Keluar dari aplikasi.
- `s`: Tukar susunan isihan proses (CPU, Memori, PID, Nama).
- `j` / `k` atau `Panah Bawah` / `Panah Atas`: Skrol senarai proses.
- `r`: Kemas kini statistik secara manual (statistik juga dikemas kini secara automatik setiap 1 saat).
