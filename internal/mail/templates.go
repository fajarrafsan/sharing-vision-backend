package mail

import "fmt"

func VerifyEmail(to, name, link string, validFor string) Message {
	return Message{
		To:      to,
		Subject: "Verifikasi email akun Warta",
		Text: fmt.Sprintf(`Halo %s,

Terima kasih sudah mendaftar di Warta. Buka tautan berikut untuk memastikan
email ini milikmu:

%s

Tautan berlaku %s. Setelah terverifikasi, kamu bisa berkomentar dan
melaporkan komentar.

Bila kamu tidak merasa mendaftar, abaikan email ini.

Warta`, name, link, validFor),
	}
}

func ResetPassword(to, name, link string, validFor string) Message {
	return Message{
		To:      to,
		Subject: "Atur ulang password Warta",
		Text: fmt.Sprintf(`Halo %s,

Ada permintaan untuk mengatur ulang password akun Warta dengan email ini.
Buka tautan berikut untuk membuat password baru:

%s

Tautan berlaku %s dan hanya bisa dipakai sekali. Setelah password diganti,
semua sesi login di perangkat lain akan keluar.

Bila kamu tidak meminta ini, abaikan email ini; password lamamu tetap berlaku.

Warta`, name, link, validFor),
	}
}
