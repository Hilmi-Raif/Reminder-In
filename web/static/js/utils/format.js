import { t, currentLang } from "../i18n/lang.js";

export function formatHumanDate(isoString) {
  const date = new Date(isoString);
  const now = new Date();

  const dDate = new Date(date.getFullYear(), date.getMonth(), date.getDate());
  const dNow = new Date(now.getFullYear(), now.getMonth(), now.getDate());

  const diffDays = Math.round((dDate - dNow) / (1000 * 60 * 60 * 24));
  const locale = typeof currentLang !== "undefined" ? currentLang : "id";
  const timeStr = date.toLocaleTimeString(locale + "-" + locale.toUpperCase(), {
    hour: "2-digit",
    minute: "2-digit",
  });

  if (diffDays === 0) return `${t("todayAt")} ${timeStr}`;
  if (diffDays === 1) return `${t("tomorrowAt")} ${timeStr}`;
  if (diffDays === -1) return `${t("yesterdayAt")} ${timeStr}`;

  const options = {
    weekday: "short",
    day: "numeric",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  };
  return date.toLocaleString(locale + "-" + locale.toUpperCase(), options);
}
