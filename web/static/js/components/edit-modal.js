import { t } from "../i18n/lang.js";
import { showMsg } from "./toast.js";
import { state, globals } from "../store/state.js";
import { loadReminders } from "./reminders-table.js";
import { htmlToWAMarkdown, formatWhatsAppMarkdown } from "../utils/html.js";
import { renderEditTargetChips } from "./target-chips.js";

const editModal = document.getElementById("edit-modal");
const closeEditBtn = document.getElementById("close-edit-btn");
const editForm = document.getElementById("edit-schedule-form");
const editRecurrenceInput = document.getElementById("edit-recurrence");
export function initEditModal() {
  window.editReminder = (id) => {
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
      ? rem.target_wa
          .split(",")
          .map((s) => s.trim())
          .filter(Boolean)
      : [];
    renderEditTargetChips();
    document.getElementById("edit-target-input").value = "";

    editRecurrenceInput.value = (rem.recurrence || "").trim();

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
      const id = document.getElementById("edit-id").value;
      const btn = document.getElementById("edit-save-btn");

      const message = globals.editQuill
        ? htmlToWAMarkdown(globals.editQuill.root.innerHTML)
        : "";
      const targetWa = (globals.editTargetNumbers || []).join(",");
      const recurrence = editRecurrenceInput.value.trim();

      if (!recurrence) {
        showMsg(t("enterCron"), true);
        return;
      }

      try {
        btn.disabled = true;
        const payload = {
          id: id,
          message: message,
          target_wa: targetWa,
          recurrence,
          scheduled_at: new Date().toISOString(),
        };

        const res = await fetch(`/api/reminders/${id}`, {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(payload),
        });

        if (res.ok) {
          editModal.classList.remove("active");
          document.body.style.overflow = "";
          showMsg(t("editSuccess"));
          state.lastETag = null;
          loadReminders(false);
        } else {
          const err = await res.text();
          showMsg(`${t("editFailed")} ${err}`, true);
        }
      } catch (err) {
        showMsg(t("editNetError"), true);
      } finally {
        btn.disabled = false;
      }
    });
}
