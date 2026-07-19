export function isValidWaFormat(val) {
  if (!val) return false;
  if (/^\d{6,15}$/.test(val)) return true;
  if (/^\d+-\d+(@g\.us)?$/.test(val)) return true;
  if (/^\d+@(s\.whatsapp\.net|g\.us|broadcast)$/.test(val)) return true;
  return false;
}

export function isValidCron(val) {
  if (!val) return false;
  const parts = val.trim().split(/\s+/);
  if (parts.length !== 5) return false;
  const cronRegex = /^[0-9*,/\-]+$/;
  return parts.every(part => cronRegex.test(part));
}

export function isMessageValid(val) {
  if (!val) return false;
  const text = val.replace(/<[^>]*>/g, '').trim();
  if (text.length === 0) return false;
  if (val.length > 4000) return false;
  return true;
}

export function isValidPhonePairingFormat(val) {
  if (!val) return false;
  const digits = val.replace(/\D/g, "");
  return digits.length >= 8 && digits.length <= 20;
}
