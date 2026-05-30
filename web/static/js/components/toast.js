import { escapeHtml } from "../utils/html.js";
import { t } from "../i18n/lang.js";

let toastIdCounter = 0;

export function showMsg(message, isError = false, durationMs = 3000) {
  const container = document.getElementById("toast-container");
  const id = ++toastIdCounter;

  const toast = document.createElement("div");
  toast.className = `toast ${isError ? "error" : "success"}`;
  toast.setAttribute("data-toast-id", id);
  toast.innerHTML = `
    <span>${escapeHtml(message)}</span>
    <button class="toast-close" title="${t("toastClose")}">&times;</button>
  `;

  const dismiss = () => {
    toast.style.animation = "toast-out 0.2s ease-in forwards";
    setTimeout(() => {
      if (toast.parentNode) toast.remove();
    }, 200);
  };

  toast.querySelector(".toast-close").addEventListener("click", dismiss);
  container.appendChild(toast);

  if (durationMs > 0) {
    setTimeout(() => {
      if (toast.parentNode) dismiss();
    }, Math.max(1000, durationMs));
  }

  return { dismiss, id };
}
