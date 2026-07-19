import { t } from "../i18n/lang.js";
import { showMsg } from "./toast.js";
import { state, globals } from "../store/state.js";
import { loadReminders } from "./reminders-table.js";
import { htmlToWAMarkdown } from "../utils/html.js";
import { renderTargetChips } from "./target-chips.js";
import { pruneMessageEditors } from "./message-editor.js";
import { createReminderApi } from "../api/reminders.js";
import { isValidCron } from "../utils/validators.js";

const scheduleForm = document.getElementById("schedule-form");
const cronInput = document.getElementById("recurrence");
const targetWaInput = document.getElementById("target-wa-input");

const sfMessageError = document.getElementById("sf-message-error");
const sfTargetError = document.getElementById("sf-target-error");
const sfRecurrenceError = document.getElementById("sf-recurrence-error");

function clearFieldErrors() {
  document.querySelectorAll(".field-error").forEach((el) => {
    if (el) el.textContent = "";
  });
}

function setFieldError(el, msg) {
  if (el) el.textContent = msg;
}

if (cronInput) {
  cronInput.addEventListener("input", () => setFieldError(sfRecurrenceError, ""));
}
if (targetWaInput) {
  targetWaInput.addEventListener("input", () => setFieldError(sfTargetError, ""));
}

export function initScheduleForm() {
  if (scheduleForm)
    scheduleForm.addEventListener("submit", async (e) => {
      e.preventDefault();
      clearFieldErrors();

      if (!state.wa_number) {
        showMsg(t("waNotLinked"), true);
        return;
      }

      const targetWa = globals.targetNumbers.join(",");
      const btn = document.getElementById("schedule-btn");
      const originalText = btn.textContent;

      let hasError = false;
      const messages = [];
      pruneMessageEditors();

      Object.keys(globals.messageEditors).forEach((id) => {
        const quill = globals.messageEditors[id];
        if (document.getElementById(`message-container-${id}`)) {
          const val = htmlToWAMarkdown(quill.root.innerHTML);
          const errEl = document.getElementById(`sf-msg-error-${id}`);
          if (!val.trim()) {
            setFieldError(errEl, t("messageEmpty"));
            hasError = true;
          } else if (val.length > 4000) {
            setFieldError(errEl, t("messageTooLong"));
            hasError = true;
          } else {
            messages.push(val);
          }
        }
      });

      if (Object.keys(globals.messageEditors).length === 0) {
        setFieldError(sfMessageError, t("enterMessage"));
        hasError = true;
      }

      const recurrence = cronInput.value.trim();
      if (!recurrence) {
        setFieldError(sfRecurrenceError, t("enterCron"));
        hasError = true;
      } else if (!isValidCron(recurrence)) {
        setFieldError(sfRecurrenceError, t("invalidCronFormat"));
        hasError = true;
      }

      if (targetWaInput && targetWaInput.value.trim()) {
        setFieldError(sfTargetError, t("targetNotAdded"));
        hasError = true;
      }

      if (hasError) return;

      try {
        btn.disabled = true;
        btn.textContent = "...";

        let successCount = 0;
        const failures = [];

        for (let i = 0; i < messages.length; i++) {
          try {
            await createReminderApi({
              message: messages[i],
              target_wa: targetWa,
              recurrence,
            });
            successCount++;
          } catch (err) {
            const reason = err.error || err.message || String(err);
            const field = err.details?.field;

            if (field) {
              if (field === "recurrence") {
                setFieldError(sfRecurrenceError, reason);
              } else if (field === "target_wa") {
                setFieldError(sfTargetError, reason);
              } else if (field === "message") {
                const activeIds = Object.keys(globals.messageEditors).filter(id => document.getElementById(`message-container-${id}`));
                const activeId = activeIds[i] || activeIds[0];
                if (activeId !== undefined) {
                  setFieldError(document.getElementById(`sf-msg-error-${activeId}`), reason);
                } else {
                  setFieldError(sfMessageError, reason);
                }
              }
              hasError = true;
              break;
            } else {
              failures.push({ error: reason });
            }
          }
        }

        if (hasError) {
          return;
        }

        if (successCount === messages.length) {
          showMsg(`${successCount} ${t("scheduled")}`);
        } else if (successCount === 0) {
          const reasons = failures.map((f) => f.error).join("; ");
          showMsg(reasons, true, 8000);
        } else {
          const reasons = failures.map((f) => f.error).join("; ");
          showMsg(`${successCount}/${messages.length} ${t("scheduled")}. ${reasons}`, true, 8000);
        }

        if (successCount > 0) {
          scheduleForm.reset();
          const messageList = document.getElementById("message-list");
          const blocks = messageList.querySelectorAll(".message-block");
          for (let i = 1; i < blocks.length; i++) blocks[i].remove();
          const removeBtn = messageList.querySelector(".remove-message-btn");
          if (removeBtn) removeBtn.style.display = "none";
          const lbl = messageList.querySelector("label");
          if (lbl) lbl.innerHTML = `<span data-i18n="messageLabel">${t("messageLabel")}</span> 1:`;
          globals.messageCount = 1;
          pruneMessageEditors();
          globals.targetNumbers = [];
          renderTargetChips();
          state.lastETag = null;
          loadReminders(true);
        }
      } catch (err) {
        showMsg(err.message || String(err), true);
      } finally {
        btn.disabled = false;
        btn.textContent = originalText || t("scheduleBtn");
      }
    });
}
