const translations = {
  en: {
    toastClose: "Close",

    darkMode: "Dark Mode",
    lightMode: "Light Mode",

    login: "Login",
    username: "Username:",
    password: "Password:",
    loginBtn: "Sign In",
    showPassword: "Show password",
    hidePassword: "Hide password",
    loginSuccess: "Logged in successfully",
    loginFailed: "Invalid username or password",
    loginError: "Failed to connect to server",
    loginRateLimited: "Too many login attempts. Try again in",
    loginRateLimitedUnknown:
      "Too many login attempts. Please wait a moment.",
    timeMinutes: "minute(s)",
    timeSeconds: "second(s)",
    logout: "Logout",
    logoutSuccess: "Logged out successfully",
    rememberMe: "Remember me",

    waConnection: "WhatsApp Connection",
    waNotConnected: "Not connected",
    waConnected: "Connected to:",
    waDisconnect: "Disconnect",
    waTabQR: "Scan QR",
    waTabPhone: "Use Phone",
    waScanQR: "Scan QR Code",
    generatingQR: "Generating QR...",
    generatingCode: "Generating...",
    qrUnavailable: "QR unavailable",
    waOrPairWith: "OR pair with:",
    waGetCode: "Get Code",
    waScanInstruction: "Scan this QR code in WhatsApp:",
    waPairInstruction:
      'Enter this pairing code under Linked Devices in WhatsApp:',
    waEnterPhone: "Enter phone number with country code",
    waInvalidPhone: "Invalid phone number format. Must be 8-20 digits.",
    waLinked: "WhatsApp connected:",
    waConnectionLost: "Connection to WhatsApp service lost",
    waUnlinked: "WhatsApp disconnected",
    waUnlinkFailed: "Failed to disconnect",
    waDisconnectTitle: "Disconnect WhatsApp?",
    waDisconnectMessage: "Re-scanning QR code will be required to reconnect.",

    scheduleReminder: "Schedule Reminder",
    messageLabel: "Message",
    removeMessage: "Remove This Message",
    addMessage: "+ Add Another Message",
    typeMessagePlaceholder: "Type a message...",
    targetWA: "Target WhatsApp (Optional):",
    targetPlaceholder: "+1234567890...",
    waPhonePlaceholder: "+1234567890...",
    addTarget: "Add",
    pickGroup: "Pick Group",
    pickContact: "Pick Contact",
    targetHint: "Leave empty to send to yourself.",
    recurrence: "Recurrence:",
    cronPlaceholder: "Cron format (e.g. */30 * * * *)",
    maxRuns: "Max Runs:",
    maxRunsPlaceholder: "0 = Unlimited",
    maxRunsHint: "Set 0 for unlimited.",
    invalidMaxRuns: "Max runs cannot be negative",
    scheduleBtn: "Schedule",

    reminderList: "Your Reminders",
    searchPlaceholder: "Search messages...",
    perPage: "/ page",
    showing: "Showing",
    of: "of",
    prev: "Prev",
    next: "Next",
    refresh: "Refresh",
    clearAll: "Clear All",
    thMessage: "Message",
    thTarget: "Target",
    thNextTime: "Scheduled For",
    thRecurrence: "Recurrence",
    thStatus: "Status",
    thActions: "Actions",
    active: "Active",
    inactive: "Inactive",
    edit: "Edit",
    delete: "Delete",
    noReminders: "No reminders found.",
    dataRefreshed: "Data refreshed",
    yourself: "Yourself",
    todayAt: "Today at",
    tomorrowAt: "Tomorrow at",
    yesterdayAt: "Yesterday at",
    sortBy: "Sort by:",
    sortMessage: "Message",
    sortTarget: "Target",
    sortTime: "Scheduled For",
    sortRecurrence: "Recurrence",

    deleteTitle: "Delete Reminder?",
    deleteMessage:
      "This reminder will be deleted permanently.",
    deleteAll: "Delete all reminders?",
    deleteAllMessage: "All reminders will be deleted permanently.",
    confirmDelete: "Yes, Delete",
    cancel: "Cancel",
    deleted: "Reminder deleted",
    allDeleted: "All reminders deleted",
    deleteFailed: "Failed to delete",

    editReminder: "Edit Reminder",
    close: "Close",
    editTargetWA: "Target WhatsApp:",
    saveChanges: "Save Changes",
    editSuccess: "Reminder updated successfully",
    editNotFound: "Reminder data not found",
    editFailed: "Failed:",
    editNetError: "Network error occurred",
    editMessageRequired: "Message is required",
    editCronRequired: "Recurrence is required",
    editLoading: "Saving...",

    waNotLinked: "Connect WhatsApp first",
    enterMessage: "Enter at least 1 message",
    enterCron: "Enter a cron expression",
    scheduled: "reminders scheduled",
    partialFail: "Some failed. Succeeded:",
    toggleFailed: "Failed to change status",
    statusActive: "Reminder activated",
    statusInactive: "Reminder deactivated",
    invalidTarget: "Invalid target",
    deleting: "Deleting...",

    invalidFormat:
      "Invalid format. Use numbers without symbols (e.g. 62812...) or Group ID.",
    invalidCronFormat: "Invalid cron format (e.g. */30 * * * *)",
    messageEmpty: "Message cannot be empty",
    messageTooLong: "Message cannot exceed 4000 characters",
    targetNotAdded: "Click Add to attach the target number",
    targetFormatError: "Invalid WhatsApp target format",
    alreadyAdded: "Already added",
    noGroupsFound: "No groups found",
    noContactsFound: "No contacts found",
    loadGroupsFailed: "Failed to load groups",
    loadContactsFailed: "Failed to load contacts",

    pick: "Pick",
    searchModalPlaceholder: "Search...",
    loading: "Loading...",

    copyright: "© 2026 ReminderIn. All Rights Reserved.",
  },
  id: {
    toastClose: "Tutup",

    darkMode: "Mode Gelap",
    lightMode: "Mode Terang",

    login: "Login",
    username: "Username:",
    password: "Password:",
    loginBtn: "Masuk",
    showPassword: "Tampilkan password",
    hidePassword: "Sembunyikan password",
    loginSuccess: "Berhasil masuk",
    loginFailed: "Username atau password salah",
    loginError: "Gagal menghubungi server",
    loginRateLimited: "Terlalu banyak percobaan login. Coba lagi dalam",
    loginRateLimitedUnknown:
      "Terlalu banyak percobaan login. Silakan tunggu beberapa saat.",
    timeMinutes: "menit",
    timeSeconds: "detik",
    logout: "Keluar",
    logoutSuccess: "Berhasil keluar",
    rememberMe: "Ingat saya",

    waConnection: "Koneksi WhatsApp",
    waNotConnected: "Belum terhubung",
    waConnected: "Tersambung ke:",
    waDisconnect: "Putuskan Koneksi",
    waTabQR: "Pindai QR",
    waTabPhone: "Gunakan Telepon",
    waScanQR: "Scan Kode QR",
    generatingQR: "Menghasilkan QR...",
    generatingCode: "Menghasilkan...",
    qrUnavailable: "QR tidak tersedia",
    waOrPairWith: "ATAU pasangkan dengan:",
    waGetCode: "Dapatkan Kode",
    waScanInstruction: "Pindai kode QR ini di WhatsApp:",
    waPairInstruction:
      'Masukkan kode pemasangan ini di menu Perangkat Tertaut pada WhatsApp:',
    waEnterPhone: "Masukkan nomor telepon dengan kode negara",
    waInvalidPhone: "Format nomor telepon tidak valid. Harus 8-20 digit.",
    waLinked: "WhatsApp tersambung:",
    waConnectionLost: "Koneksi ke layanan WhatsApp terputus",
    waUnlinked: "WhatsApp diputuskan",
    waUnlinkFailed: "Gagal memutuskan",
    waDisconnectTitle: "Putuskan WhatsApp?",
    waDisconnectMessage: "Diperlukan pemindaian ulang kode QR untuk menyambungkan kembali.",

    scheduleReminder: "Jadwalkan Pengingat",
    messageLabel: "Pesan",
    removeMessage: "Hapus Pesan Ini",
    addMessage: "+ Tambah Pesan Lain",
    typeMessagePlaceholder: "Ketik pesan...",
    targetWA: "Target WhatsApp (Opsional):",
    targetPlaceholder: "+628123...",
    waPhonePlaceholder: "+628123...",
    addTarget: "Tambah",
    pickGroup: "Pilih Grup",
    pickContact: "Pilih Kontak",
    targetHint: "Kosongkan untuk kirim ke diri sendiri.",
    recurrence: "Pengulangan:",
    cronPlaceholder: "Format cron (misal: */30 * * * *)",
    maxRuns: "Batas Pengulangan:",
    maxRunsPlaceholder: "0 = Tanpa batas",
    maxRunsHint: "Isi 0 untuk tanpa batas.",
    invalidMaxRuns: "Batas pengulangan tidak boleh negatif",
    scheduleBtn: "Jadwalkan",

    reminderList: "Daftar Pengingat Anda",
    searchPlaceholder: "Cari pesan...",
    perPage: "/ hal",
    showing: "Menampilkan",
    of: "dari",
    prev: "Seb.",
    next: "Selanj.",
    refresh: "Refresh",
    clearAll: "Hapus Semua",
    thMessage: "Pesan",
    thTarget: "Target",
    thNextTime: "Dijadwalkan Untuk",
    thRecurrence: "Pengulangan",
    thStatus: "Status",
    thActions: "Aksi",
    active: "Aktif",
    inactive: "Nonaktif",
    edit: "Edit",
    delete: "Hapus",
    noReminders: "Tidak ada pengingat.",
    dataRefreshed: "Data dimuat ulang",
    yourself: "Diri Sendiri",
    todayAt: "Hari ini pukul",
    tomorrowAt: "Besok pukul",
    yesterdayAt: "Kemarin pukul",
    sortBy: "Urutkan:",
    sortMessage: "Pesan",
    sortTarget: "Target",
    sortTime: "Dijadwalkan Untuk",
    sortRecurrence: "Pengulangan",

    deleteTitle: "Hapus Pengingat?",
    deleteMessage:
      "Pengingat ini akan dihapus secara permanen.",
    deleteAll: "Hapus semua pengingat?",
    deleteAllMessage:
      "Semua pengingat akan dihapus secara permanen.",
    confirmDelete: "Ya, Hapus",
    cancel: "Batal",
    deleted: "Pengingat dihapus",
    allDeleted: "Semua pengingat dihapus",
    deleteFailed: "Gagal menghapus",

    editReminder: "Edit Pengingat",
    close: "Tutup",
    editTargetWA: "Target WhatsApp:",
    saveChanges: "Simpan Perubahan",
    editSuccess: "Pengingat berhasil diperbarui",
    editNotFound: "Data pengingat tidak ditemukan",
    editFailed: "Gagal:",
    editNetError: "Terjadi kesalahan jaringan",
    editMessageRequired: "Pesan wajib diisi",
    editCronRequired: "Pengulangan wajib diisi",
    editLoading: "Menyimpan...",

    waNotLinked: "Sambungkan WhatsApp terlebih dahulu",
    enterMessage: "Masukkan setidaknya 1 pesan",
    enterCron: "Masukkan cron expression",
    scheduled: "pengingat dijadwalkan",
    partialFail: "Beberapa gagal. Berhasil:",
    toggleFailed: "Gagal mengubah status",
    statusActive: "Pengingat diaktifkan",
    statusInactive: "Pengingat dinonaktifkan",
    invalidTarget: "Target tidak valid",
    deleting: "Menghapus...",

    invalidFormat:
      "Format tidak valid. Gunakan angka tanpa simbol (contoh: 62812...) atau ID Grup.",
    invalidCronFormat: "Format cron tidak valid (misal: */30 * * * *)",
    messageEmpty: "Pesan tidak boleh kosong",
    messageTooLong: "Pesan maksimal 4000 karakter",
    targetNotAdded: "Klik Tambah untuk memasukkan nomor target",
    targetFormatError: "Format nomor WA tidak valid",
    alreadyAdded: "Sudah ditambahkan",
    noGroupsFound: "Tidak ada grup ditemukan",
    noContactsFound: "Tidak ada kontak ditemukan",
    loadGroupsFailed: "Gagal memuat grup",
    loadContactsFailed: "Gagal memuat kontak",

    pick: "Pilih",
    searchModalPlaceholder: "Cari...",
    loading: "Memuat...",

    copyright: "© 2026 ReminderIn. Hak Cipta Dilindungi.",
  },
};

export let currentLang = localStorage.getItem("rm_lang") || "en";

export function t(key) {
  return (
    (translations[currentLang] && translations[currentLang][key]) ||
    translations.en[key] ||
    key
  );
}

export function applyLanguage(lang) {
  currentLang = lang;
  localStorage.setItem("rm_lang", lang);

  document.querySelectorAll("[data-i18n]").forEach((el) => {
    el.textContent = t(el.dataset.i18n);
  });

  document.querySelectorAll("[data-i18n-placeholder]").forEach((el) => {
    el.placeholder = t(el.dataset.i18nPlaceholder);
  });

  document.querySelectorAll("[data-i18n-title]").forEach((el) => {
    el.title = t(el.dataset.i18nTitle);
  });

  document.querySelectorAll(".lite-content[data-placeholder]").forEach((el) => {
    el.setAttribute("data-placeholder", t("typeMessagePlaceholder"));
  });

  const langBtn = document.getElementById("lang-toggle");
  if (langBtn) langBtn.textContent = lang === "en" ? "ID" : "EN";

  const themeBtn = document.getElementById("theme-toggle");
  if (themeBtn) {
    const isDark =
      document.documentElement.getAttribute("data-theme") === "dark";
    themeBtn.textContent = isDark ? t("lightMode") : t("darkMode");
  }

  document.querySelectorAll("#page-size option").forEach((opt) => {
    opt.textContent = opt.value + " " + t("perPage");
  });

  if (!document.getElementById("app-view").hidden) {
    import("../components/wa-connection.js")
      .then((m) => {
        if (typeof m.updateDash === "function") m.updateDash();
      })
      .catch(() => {});

    import("../components/reminders-table.js")
      .then((m) => {
        if (typeof m.rerenderRemindersLocale === "function")
          m.rerenderRemindersLocale();
      })
      .catch(() => {});
  }
}

document.addEventListener("DOMContentLoaded", () => {
  const langBtn = document.getElementById("lang-toggle");
  if (langBtn) {
    langBtn.textContent = currentLang === "en" ? "ID" : "EN";
    langBtn.addEventListener("click", () => {
      applyLanguage(currentLang === "en" ? "id" : "en");
    });
  }
});
