import { t } from "../i18n/lang.js";
import { showMsg } from "./toast.js";
import { state, globals } from "../store/state.js";
import { showLogin } from "./auth.js";
import { createQrEventSource } from "../api/whatsapp.js";
import { isValidPhonePairingFormat } from "../utils/validators.js";

const waStatus = document.getElementById("wa-status");
const waLinkActions = document.getElementById("wa-link-actions");
const linkWaQrBtn = document.getElementById("link-wa-qr-btn");
const linkWaPhoneBtn = document.getElementById("link-wa-phone-btn");
const waPhoneInput = document.getElementById("wa-phone");
const qrContainer = document.getElementById("qr-container");
const qrImg = document.getElementById("qr-img");
const codeContainer = document.getElementById("code-container");
const pairCodeDisplay = document.getElementById("pair-code-display");
const unlinkWaBtn = document.getElementById("unlink-wa-btn");
const pickGroupBtn = document.getElementById("pick-group-btn");
const pickContactBtn = document.getElementById("pick-contact-btn");

let activeQrEvtSource = null;
let activePhoneEvtSource = null;

export function updateDash() {
  const editPickGroupBtn = document.getElementById("edit-pick-group-btn");
  const editPickContactBtn = document.getElementById("edit-pick-contact-btn");
  if (state.wa_number) {
    waStatus.textContent = `${t("waConnected")} ${state.wa_number}`;
    waLinkActions.hidden = true;
    qrContainer.hidden = true;
    codeContainer.hidden = true;
    if (pickGroupBtn) pickGroupBtn.disabled = false;
    if (pickContactBtn) pickContactBtn.disabled = false;
    if (editPickGroupBtn) editPickGroupBtn.disabled = false;
    if (editPickContactBtn) editPickContactBtn.disabled = false;
    if (unlinkWaBtn) unlinkWaBtn.hidden = false;
  } else {
    waStatus.textContent = t("waNotConnected");
    waLinkActions.hidden = false;
    if (pickGroupBtn) pickGroupBtn.disabled = true;
    if (pickContactBtn) pickContactBtn.disabled = true;
    if (editPickGroupBtn) editPickGroupBtn.disabled = true;
    if (editPickContactBtn) editPickContactBtn.disabled = true;
    if (unlinkWaBtn) unlinkWaBtn.hidden = true;
  }
}

export async function initWA() {
  try {
    const res = await fetch("/api/wa/status");
    if (res.status === 401) {
      showLogin();
      return;
    }
    if (res.ok) {
      const data = await res.json();
      if (data.status === "connected") {
        state.wa_number = data.number;
        localStorage.setItem("rm_wa_number", data.number);
      } else {
        state.wa_number = null;
        localStorage.removeItem("rm_wa_number");
      }
    }
  } catch (e) {
    console.error("Failed to get WA status", e);
  }
  updateDash();
}

export function initWaConnection() {
  const tabQr = document.getElementById("tab-qr");
  const tabPhone = document.getElementById("tab-phone");
  const viewQr = document.getElementById("view-qr");
  const viewPhone = document.getElementById("view-phone");

  if (tabQr && tabPhone) {
    tabQr.addEventListener("click", () => {
      tabQr.classList.add("active");
      tabPhone.classList.remove("active");

      viewQr.classList.add("active");
      viewPhone.classList.remove("active");

      if (activePhoneEvtSource) {
        activePhoneEvtSource.close();
        activePhoneEvtSource = null;
      }
      codeContainer.hidden = true;
      linkWaPhoneBtn.disabled = false;
    });

    tabPhone.addEventListener("click", () => {
      tabPhone.classList.add("active");
      tabQr.classList.remove("active");

      viewPhone.classList.add("active");
      viewQr.classList.remove("active");

      if (activeQrEvtSource) {
        activeQrEvtSource.close();
        activeQrEvtSource = null;
      }
      qrContainer.hidden = true;
      linkWaQrBtn.disabled = false;
    });
  }

  if (linkWaQrBtn)
    linkWaQrBtn.addEventListener("click", () => {
      qrContainer.hidden = false;
      codeContainer.hidden = true;
      qrImg.src = "";
      qrImg.alt = "Menghasilkan QR...";
      linkWaQrBtn.disabled = true;

      if (activeQrEvtSource) {
        activeQrEvtSource.close();
      }
      activeQrEvtSource = createQrEventSource();
      const evtSource = activeQrEvtSource;

      evtSource.onmessage = (e) => {
        const data = JSON.parse(e.data);
        if (data.type === "qr") {
          if (data.image) {
            qrImg.src = data.image;
            qrImg.alt = "QR Code";
          } else {
            qrImg.src = "";
            qrImg.alt = "QR unavailable";
          }
          linkWaQrBtn.disabled = false;
        } else if (data.type === "success") {
          state.wa_number = data.number;
          localStorage.setItem("rm_wa_number", data.number);
          showMsg(`${t("waLinked")} ${data.number}`);
          evtSource.close();
          updateDash();
        } else if (data.type === "error") {
          showMsg(data.message, true);
          evtSource.close();
          linkWaQrBtn.disabled = false;
          qrContainer.hidden = true;
        }
      };

      evtSource.onerror = () => {
        showMsg(t("waConnectionLost"), true);
        evtSource.close();
        linkWaQrBtn.disabled = false;
        qrContainer.hidden = true;
      };
    });

  if (linkWaPhoneBtn) {
    const waPhoneError = document.getElementById("wa-phone-error");

    if (waPhoneInput) {
      waPhoneInput.addEventListener("input", () => {
        if (waPhoneError) waPhoneError.textContent = "";
      });
    }

    linkWaPhoneBtn.addEventListener("click", () => {
      if (waPhoneError) waPhoneError.textContent = "";
      const phone = waPhoneInput.value.trim();
      if (!phone) {
        if (waPhoneError) waPhoneError.textContent = t("waEnterPhone");
        return;
      }
      if (!isValidPhonePairingFormat(phone)) {
        if (waPhoneError) waPhoneError.textContent = t("waInvalidPhone");
        return;
      }

      codeContainer.hidden = false;
      qrContainer.hidden = true;
      pairCodeDisplay.textContent = "Menghasilkan...";
      linkWaPhoneBtn.disabled = true;

      if (activePhoneEvtSource) {
        activePhoneEvtSource.close();
      }
      activePhoneEvtSource = new EventSource(
        `/api/wa/pair?phone=${encodeURIComponent(phone)}`,
      );
      const evtSource = activePhoneEvtSource;

      evtSource.onmessage = (e) => {
        const data = JSON.parse(e.data);
        if (data.type === "code") {
          pairCodeDisplay.textContent = data.code;
          linkWaPhoneBtn.disabled = false;
        } else if (data.type === "success") {
          state.wa_number = data.number;
          localStorage.setItem("rm_wa_number", data.number);
          showMsg(`${t("waLinked")} ${data.number}`);
          evtSource.close();
          updateDash();
        } else if (data.type === "error") {
          if (waPhoneError) waPhoneError.textContent = data.message;
          evtSource.close();
          linkWaPhoneBtn.disabled = false;
          codeContainer.hidden = true;
        }
      };

      evtSource.onerror = () => {
        showMsg(t("waConnectionLost"), true);
        evtSource.close();
        linkWaPhoneBtn.disabled = false;
        codeContainer.hidden = true;
      };
    });
  }

  if (unlinkWaBtn)
    unlinkWaBtn.addEventListener("click", () => {
      const disconnectModal = document.getElementById("disconnect-modal");
      disconnectModal.classList.add("active");
      document.body.style.overflow = "hidden";
    });

  const confirmDisconnectBtn = document.getElementById("confirm-disconnect-btn");
  const cancelDisconnectBtn = document.getElementById("cancel-disconnect-btn");
  const disconnectModal = document.getElementById("disconnect-modal");

  if (confirmDisconnectBtn)
    confirmDisconnectBtn.addEventListener("click", async () => {
      confirmDisconnectBtn.disabled = true;
      try {
        const res = await fetch("/api/wa", { method: "DELETE" });
        if (res.ok) {
          state.wa_number = null;
          localStorage.removeItem("rm_wa_number");
          updateDash();
          showMsg(t("waUnlinked"));
        }
      } catch (err) {
        showMsg(t("waUnlinkFailed"), true);
      } finally {
        confirmDisconnectBtn.disabled = false;
        disconnectModal.classList.remove("active");
        document.body.style.overflow = "";
      }
    });

  if (cancelDisconnectBtn)
    cancelDisconnectBtn.addEventListener("click", () => {
      disconnectModal.classList.remove("active");
      document.body.style.overflow = "";
    });

  if (disconnectModal)
    disconnectModal.addEventListener("click", (e) => {
      if (e.target === disconnectModal) cancelDisconnectBtn.click();
    });
}
