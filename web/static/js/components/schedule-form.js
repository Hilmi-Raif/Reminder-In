import { t } from "../i18n/lang.js";
import { showMsg } from "./toast.js";
import { state, globals } from "../store/state.js";
import { loadReminders } from "./reminders-table.js";
import { htmlToWAMarkdown } from "../utils/html.js";
import { renderTargetChips } from "./target-chips.js";
import { pruneMessageEditors } from "./message-editor.js";

const scheduleForm = document.getElementById("schedule-form");
const cronInput = document.getElementById("recurrence");
export function initScheduleForm() {
  if (scheduleForm)
    scheduleForm.addEventListener("submit", async (e) => {
      e.preventDefault();
      if (!state.wa_number) {
        showMsg(t("waNotLinked"), true);
        return;
      }

      const targetWa = globals.targetNumbers.join(",");
      const btn = document.getElementById("schedule-btn");

      try {
        btn.disabled = true;
        const messages = [];

        pruneMessageEditors();

        Object.keys(globals.messageEditors).forEach((id) => {
          const quill = globals.messageEditors[id];

          if (document.getElementById(`message-container-${id}`)) {
            const val = htmlToWAMarkdown(quill.root.innerHTML);
            if (val !== "") {
              messages.push(val);
            }
          }
        });

        if (messages.length === 0) {
          showMsg(t("enterMessage"), true);
          btn.disabled = false;
          return;
        }

        const recurrence = cronInput.value.trim();
        if (!recurrence) {
          showMsg(t("enterCron"), true);
          btn.disabled = false;
          return;
        }

        const isoDate = new Date().toISOString();

        let successCount = 0;
        for (const msg of messages) {
          const res = await fetch("/api/reminders", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
              message: msg,
              target_wa: targetWa,
              recurrence,
              scheduled_at: isoDate,
            }),
          });
          if (res.ok) successCount++;
        }

        if (successCount === messages.length) {
          showMsg(`${successCount} ${t("scheduled")}`);
        } else {
          showMsg(
            `${t("partialFail")} ${successCount}/${messages.length}`,
            true,
          );
        }

        scheduleForm.reset();

        const messageList = document.getElementById("message-list");
        const blocks = messageList.querySelectorAll(".message-block");
        for (let i = 1; i < blocks.length; i++) {
          blocks[i].remove();
        }
        const removeBtn = messageList.querySelector(".remove-message-btn");
        if (removeBtn) removeBtn.style.display = "none";
        const lbl = messageList.querySelector("label");
        if (lbl)
          lbl.innerHTML = `<span data-i18n="messageLabel">${t("messageLabel")}</span> 1:`;
        globals.messageCount = 1;
        pruneMessageEditors();

        globals.targetNumbers = [];
        renderTargetChips();
        state.lastETag = null;
        loadReminders(true);
      } catch (err) {
        showMsg(err.message, true);
      } finally {
        btn.disabled = false;
      }
    });
}
