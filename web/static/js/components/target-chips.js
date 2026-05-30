import { t } from "../i18n/lang.js";
import { escapeHtml } from "../utils/html.js";
import { showMsg } from "./toast.js";
import { globals } from "../store/state.js";
import { isValidWaFormat } from "../utils/validators.js";

const targetWaInput = document.getElementById("target-wa-input");
const addTargetBtn = document.getElementById("add-target-btn");
const targetChips = document.getElementById("target-chips");

const editAddTargetBtn = document.getElementById("edit-add-target-btn");
const editTargetInput = document.getElementById("edit-target-input");

const sfTargetError = document.getElementById("sf-target-error");
const emTargetError = document.getElementById("em-target-error");

export function renderTargetChips() {
  if (!targetChips) return;
  targetChips.innerHTML = "";
  globals.targetNumbers.forEach((num, idx) => {
    const chip = document.createElement("span");
    chip.className = "chip";
    const label =
      globals.groupsCache && globals.groupsCache[num]
        ? globals.groupsCache[num]
        : globals.contactsCache && globals.contactsCache[num]
          ? globals.contactsCache[num]
          : num;
    chip.innerHTML = `${escapeHtml(label)}<button type="button" onclick="removeTarget(${idx})">&times;</button>`;
    targetChips.appendChild(chip);
  });
}

export function renderEditTargetChips() {
  const container = document.getElementById("edit-target-chips");
  if (!container) return;
  container.innerHTML = "";
  globals.editTargetNumbers.forEach((num, idx) => {
    const chip = document.createElement("span");
    chip.className = "chip";
    const label =
      globals.groupsCache && globals.groupsCache[num]
        ? globals.groupsCache[num]
        : globals.contactsCache && globals.contactsCache[num]
          ? globals.contactsCache[num]
          : num;
    chip.innerHTML = `${escapeHtml(label)}<button type="button" onclick="removeEditTarget(${idx})">&times;</button>`;
    container.appendChild(chip);
  });
}

export function initTargetChips() {
  if (addTargetBtn)
    addTargetBtn.addEventListener("click", () => {
      if (sfTargetError) sfTargetError.textContent = "";
      const val = targetWaInput.value.trim();
      if (!val) return;

      if (!isValidWaFormat(val)) {
        if (sfTargetError) sfTargetError.textContent = t("targetFormatError");
        return;
      }

      if (globals.targetNumbers.includes(val)) {
        if (sfTargetError) sfTargetError.textContent = t("alreadyAdded");
        return;
      }
      globals.targetNumbers.push(val);
      renderTargetChips();
      targetWaInput.value = "";
      targetWaInput.focus();
    });

  if (targetWaInput) {
    targetWaInput.addEventListener("keydown", (e) => {
      if (e.key === "Enter") {
        e.preventDefault();
        addTargetBtn.click();
      }
    });
    targetWaInput.addEventListener("input", () => {
      if (sfTargetError) sfTargetError.textContent = "";
    });
  }

  window.removeTarget = (idx) => {
    globals.targetNumbers.splice(idx, 1);
    renderTargetChips();
    if (sfTargetError) sfTargetError.textContent = "";
  };

  if (editAddTargetBtn)
    editAddTargetBtn.addEventListener("click", () => {
      if (emTargetError) emTargetError.textContent = "";
      const val = editTargetInput.value.trim();
      if (!val) return;

      if (!isValidWaFormat(val)) {
        if (emTargetError) emTargetError.textContent = t("targetFormatError");
        return;
      }

      if (globals.editTargetNumbers.includes(val)) {
        if (emTargetError) emTargetError.textContent = t("alreadyAdded");
        return;
      }
      globals.editTargetNumbers.push(val);
      renderEditTargetChips();
      editTargetInput.value = "";
      editTargetInput.focus();
    });

  if (editTargetInput) {
    editTargetInput.addEventListener("keydown", (e) => {
      if (e.key === "Enter") {
        e.preventDefault();
        editAddTargetBtn.click();
      }
    });
    editTargetInput.addEventListener("input", () => {
      if (emTargetError) emTargetError.textContent = "";
    });
  }

  window.removeEditTarget = (idx) => {
    globals.editTargetNumbers.splice(idx, 1);
    renderEditTargetChips();
    if (emTargetError) emTargetError.textContent = "";
  };
}
