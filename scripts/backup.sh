#!/usr/bin/env bash
# Backup database dan gambar yang diunggah.
#
#   backup.sh once    satu kali backup lalu selesai
#   backup.sh loop    backup sekarang, lalu setiap BACKUP_INTERVAL_HOURS jam
#   backup.sh check   berhasil bila backup terakhir cukup baru (healthcheck)
#   backup.sh restore <folder>   memulihkan, lihat restore.sh
#
# Setiap backup adalah satu folder BACKUP_DIR/warta-<waktu UTC> berisi
# database.sql.gz, uploads.tar.gz (bila UPLOAD_DIR ada), dan SHA256SUMS.
# Folder dikerjakan dengan akhiran .partial lalu di-rename setelah lengkap,
# jadi folder tanpa akhiran itu selalu utuh. Backup yang lebih tua dari
# BACKUP_KEEP_DAYS hari dihapus.
set -euo pipefail

: "${DB_HOST:=127.0.0.1}"
: "${DB_PORT:=3306}"
: "${DB_USER:=root}"
: "${DB_PASSWORD:=}"
: "${DB_NAME:=warta}"
: "${BACKUP_DIR:=/backups}"
: "${UPLOAD_DIR:=}"
: "${BACKUP_KEEP_DAYS:=14}"
: "${BACKUP_INTERVAL_HOURS:=24}"

log() {
	printf '%s backup: %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*"
}

backup_once() {
	local stamp target work
	stamp="$(date -u +%Y%m%dT%H%M%SZ)"
	target="${BACKUP_DIR}/warta-${stamp}"
	work="${target}.partial"
	mkdir -p "$work"

	log "mulai ${target}"

	# --single-transaction membaca snapshot yang konsisten tanpa mengunci
	# tabel InnoDB, jadi service tetap berjalan. --no-tablespaces supaya user
	# biasa (bukan root) cukup. Password lewat MYSQL_PWD agar tidak terlihat
	# di daftar proses.
	MYSQL_PWD="$DB_PASSWORD" mysqldump \
		--host="$DB_HOST" --port="$DB_PORT" --user="$DB_USER" \
		--single-transaction --quick --routines --triggers \
		--no-tablespaces --set-gtid-purged=OFF \
		--default-character-set=utf8mb4 \
		"$DB_NAME" | gzip -9 >"${work}/database.sql.gz"

	if [[ -n "$UPLOAD_DIR" && -d "$UPLOAD_DIR" ]]; then
		tar -czf "${work}/uploads.tar.gz" -C "$UPLOAD_DIR" .
	else
		log "UPLOAD_DIR kosong atau tidak ada, gambar tidak ikut (object storage punya backup sendiri)"
	fi

	# Pastikan arsip bisa dibaca sebelum dianggap berhasil.
	gzip -t "${work}/database.sql.gz"
	if [[ -f "${work}/uploads.tar.gz" ]]; then
		tar -tzf "${work}/uploads.tar.gz" >/dev/null
	fi
	(cd "$work" && sha256sum -- *.gz >SHA256SUMS)

	mv "$work" "$target"
	touch "${BACKUP_DIR}/.last-success"
	log "selesai ${target} ($(du -sh "$target" | cut -f1))"

	prune
}

prune() {
	# Backup lama dan sisa backup yang gagal di tengah jalan.
	find "$BACKUP_DIR" -mindepth 1 -maxdepth 1 -type d -name 'warta-*' \
		-mtime "+${BACKUP_KEEP_DAYS}" -print -exec rm -rf {} + |
		while read -r old; do log "hapus ${old}"; done
	find "$BACKUP_DIR" -mindepth 1 -maxdepth 1 -type d -name 'warta-*.partial' \
		-mmin +60 -exec rm -rf {} +
}

check() {
	# Sehat bila backup terakhir tidak lebih tua dari dua kali interval.
	local limit=$((BACKUP_INTERVAL_HOURS * 120))
	[[ -n "$(find "$BACKUP_DIR" -maxdepth 1 -name .last-success -mmin "-${limit}" 2>/dev/null)" ]]
}

mkdir -p "$BACKUP_DIR"

case "${1:-once}" in
once)
	backup_once
	;;
loop)
	while true; do
		# Kegagalan satu kali dicatat, lalu dicoba lagi pada jadwal berikutnya.
		backup_once || log "GAGAL, dicoba lagi dalam ${BACKUP_INTERVAL_HOURS} jam"
		sleep "$((BACKUP_INTERVAL_HOURS * 3600))"
	done
	;;
check)
	check
	;;
restore)
	exec bash "$(dirname "$0")/restore.sh" "${2:-}"
	;;
*)
	echo "pemakaian: $0 once|loop|check|restore <folder>" >&2
	exit 2
	;;
esac
