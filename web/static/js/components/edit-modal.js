import { t } from "../i18n/lang.js";
import { showMsg } from "./toast.js";
import { state, globals } from "../store/state.js";
import { loadReminders } from "./reminders-table.js";
import { htmlToWAMarkdown, formatWhatsAppMarkdown } from "../utils/html.js";
import { renderEditTargetChips } from "./target-chips.js";
import { updateReminderApi } from "../api/reminders.js";
import { isValidCron } from "../utils/validators.js";

const editModal = document.getElementById("edit-modal");
const closeEditBtn = document.getElementById("close-edit-btn");
const editForm = document.getElementById("edit-schedule-form");
const editRecurrenceInput = document.getElementById("edit-recurrence");
const editTargetInput = document.getElementById("edit-target-input");

const emMessageError = document.getElementById("em-message-error");
const emTargetError = document.getElementById("em-target-error");
const emRecurrenceError = document.getElementById("em-recurrence-error");

function clearEditErrors() {
  [emMessageError, emTargetError, emRecurrenceError].forEach((el) => {
    if (el) el.textContent = "";
  });
}

function setEditFieldError(el, msg) {
  if (el) el.textContent = msg;
}

if (editRecurrenceInput) {
  editRecurrenceInput.addEventListener("input", () => setEditFieldError(emRecurrenceError, ""));
}
if (editTargetInput) {
  editTargetInput.addEventListener("input", () => setEditFieldError(emTargetError, ""));
}

export function initEditModal() {
  window.editReminder = (id) => {
    clearEditErrors();

    const rem = state.remindersData.find((r) => r.id === id);
    if (!rem) {
      showMsg(t("editNotFound"), true);
      return;
    }

    document.getElementById("edit-id").value = rem.id;
    if (globals.editQuill) {
      globals.editQuill.root.innerHTML = formatWhatsAppMarkdown(rem.message);
    }

    globals.editTargetNumbers = rem.target_wa
      ? rem.target_wa.split(",").map((s) => s.trim()).filter(Boolean)
      : [];
    renderEditTargetChips();
    document.getElementById("edit-target-input").value = "";

    editRecurrenceInput.value = (rem.recurrence || "").trim();
    globals.editOriginalScheduledAt = rem.scheduled_at;

    editModal.classList.add("active");
    document.body.style.overflow = "hidden";
  };

  if (closeEditBtn)
    closeEditBtn.addEventListener("click", () => {
      editModal.classList.remove("active");
      document.body.style.overflow = "";
    });

  if (editModal)
    editModal.addEventListener("click", (e) => {
      if (e.target === editModal) closeEditBtn.click();
    });

  if (editForm)
    editForm.addEventListener("submit", async (e) => {
      e.preventDefault();
      clearEditErrors();

      const id = document.getElementById("edit-id").value;
      const btn = document.getElementById("edit-save-btn");
      const originalText = btn.textContent;

      const message = globals.editQuill
        ? htmlToWAMarkdown(globals.editQuill.root.innerHTML)
        : "";
      const targetWa = (globals.editTargetNumbers || []).join(",");
      const recurrence = editRecurrenceInput.value.trim();

      let hasError = false;
      if (!message) {
        setEditFieldError(emMessageError, t("messageEmpty"));
        hasError = true;
      } else if (message.length > 4000) {
        setEditFieldError(emMessageError, t("messageTooLong"));
        hasError = true;
      }

      if (!recurrence) {
        setEditFieldError(emRecurrenceError, t("enterCron"));
        hasError = true;
      } else if (!isValidCron(recurrence)) {
        setEditFieldError(emRecurrenceError, t("invalidCronFormat"));
        hasError = true;
      }

      if (editTargetInput && editTargetInput.value.trim()) {
        setEditFieldError(emTargetError, t("targetNotAdded"));
        hasError = true;
      }

      if (hasError) return;

      try {
        btn.disabled = true;
        btn.textContent = t("editLoading");

        const scheduledAt = globals.editOriginalScheduledAt || new Date().toISOString();

        const payload = {
          id: id,
          message: message,
          target_wa: targetWa,
          recurrence,
          scheduled_at: scheduledAt,
        };

        await updateReminderApi(id, payload);

        editModal.classList.remove("active");
        document.body.style.overflow = "";
        showMsg(t("editSuccess"));
        state.lastETag = null;
        loadReminders(false);
      } catch (err) {
        const reason = err.error || err.message || t("editNetError");
        const field = err.details?.field;

        if (field) {
          if (field === "recurrence") {
            setEditFieldError(emRecurrenceError, reason);
          } else if (field === "target_wa") {
            setEditFieldError(emTargetError, reason);
          } else if (field === "message") {
            setEditFieldError(emMessageError, reason);
          } else if (field === "scheduled_at") {
            setEditFieldError(emRecurrenceError, reason);
          }
        } else {
          showMsg(reason, true);
        }
      } finally {
        btn.disabled = false;
        btn.textContent = originalText || t("saveChanges");
      }
    });
}
