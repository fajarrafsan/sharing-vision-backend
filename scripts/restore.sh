#!/usr/bin/env bash
# Mengembalikan database dan gambar dari satu folder backup buatan backup.sh.
#
#   RESTORE_CONFIRM=yes restore.sh /backups/warta-20260929T020000Z
#
# Isi database DB_NAME diganti seluruhnya dengan isi backup. Hentikan service
# API dulu supaya tidak ada tulisan baru selama restore. Gambar di UPLOAD_DIR
# ditambahkan dari arsip; berkas yang sudah ada tidak dihapus.
set -euo pipefail

: "${DB_HOST:=127.0.0.1}"
: "${DB_PORT:=3306}"
: "${DB_USER:=root}"
: "${DB_PASSWORD:=}"
: "${DB_NAME:=warta}"
: "${UPLOAD_DIR:=}"

source_dir="${1:-}"
if [[ -z "$source_dir" || ! -f "${source_dir}/database.sql.gz" ]]; then
	echo "pemakaian: RESTORE_CONFIRM=yes $0 <folder backup berisi database.sql.gz>" >&2
	exit 2
fi
if [[ "${RESTORE_CONFIRM:-}" != "yes" ]]; then
	echo "restore mengganti seluruh isi database ${DB_NAME} di ${DB_HOST}." >&2
	echo "jalankan ulang dengan RESTORE_CONFIRM=yes bila sudah yakin." >&2
	exit 1
fi

log() {
	printf '%s restore: %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*"
}

log "memeriksa checksum ${source_dir}"
(cd "$source_dir" && sha256sum -c --quiet SHA256SUMS)

log "memulihkan database ${DB_NAME}"
MYSQL_PWD="$DB_PASSWORD" mysql --host="$DB_HOST" --port="$DB_PORT" --user="$DB_USER" \
	--default-character-set=utf8mb4 \
	-e "CREATE DATABASE IF NOT EXISTS \`${DB_NAME}\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"
gzip -dc "${source_dir}/database.sql.gz" |
	MYSQL_PWD="$DB_PASSWORD" mysql --host="$DB_HOST" --port="$DB_PORT" --user="$DB_USER" \
		--default-character-set=utf8mb4 "$DB_NAME"

if [[ -f "${source_dir}/uploads.tar.gz" ]]; then
	if [[ -n "$UPLOAD_DIR" ]]; then
		log "memulihkan gambar ke ${UPLOAD_DIR}"
		mkdir -p "$UPLOAD_DIR"
		tar -xzf "${source_dir}/uploads.tar.gz" -C "$UPLOAD_DIR"
	else
		log "UPLOAD_DIR kosong, gambar dilewati"
	fi
fi

log "selesai"
