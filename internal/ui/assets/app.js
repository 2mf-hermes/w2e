// w2e desktop UI controller — vanilla JS, no framework, no CDN.
//
// Guided wizard: one step visible at a time, prev/next navigation,
// validation per step, and build execution on the last step.
(() => {
  "use strict";

  // ===== frontend error capture =====
  function report(level, msg, source, line, col) {
    try {
      fetch("/api/log", {
        method: "POST",
        headers: { "Content-Type": "application/json; charset=utf-8" },
        body: JSON.stringify({ level, msg: String(msg), source: String(source || ""),
                               line: line | 0, col: col | 0 }),
      }).catch(function () {});
    } catch (_) {}
  }
  window.addEventListener("error", (e) => {
    report("error", e.message, e.filename, e.lineno, e.colno);
  });
  window.addEventListener("unhandledrejection", (e) => {
    report("promise", (e.reason && e.reason.stack) || String(e.reason), "", 0, 0);
  });
  const w2eLog = { report };

  const $ = (sel, root = document) => root.querySelector(sel);
  const $$ = (sel, root = document) => Array.from(root.querySelectorAll(sel));

  // ===== Element refs =====
  const els = {
    langWrap: $("#locale-dropdown"),
    langBtn: $(".apple-dropdown-trigger"),
    langMenu: $(".apple-dropdown-menu"),
    langInput: $("#locale-select"),
    form: $("#build-form"),
    sourceDir: $("#source-dir"),
    output: $("#output"),
    icon: $("#icon"),
    width: $("#width"),
    height: $("#height"),
    title: $("#window-title"),
    resizable: $("#resizable"),
    targetGroup: $("#target-group"),
    validate: $("#validate-btn"),
    start: $("#start-btn"),
    status: $("#status"),
    report: $("#report"),
    appVersion: $("#app-version"),
    themeSlider: $("#theme-slider"),
    sliderPill: $("#theme-slider .slider-pill"),
    themeBtns: $$(".theme-slider-item"),
    // Wizard
    steps: $$(".wizard-step"),
    dots: $$(".wiz-dot"),
    lines: $$(".wiz-line"),
    prevBtn: $("#wiz-prev"),
    nextBtn: $("#wiz-next"),
    indicator: $(".wizard-indicator"),
  };

  const cache = { i18n: {} };
  const TOTAL_STEPS = 4;
  let currentStep = 1;
  let validated = false; // true after successful validate

  // ===== Helper: reliably hide/show elements =====
  function hideEl(el) { if (el) { el.style.display = "none"; el.setAttribute("aria-hidden", "true"); } }
  function showEl(el)  { if (el) { el.style.display = ""; el.removeAttribute("aria-hidden"); } }

  // ===== i18n =====
  async function loadLocale(code) {
    try {
      const res = await fetch(`/api/i18n?locale=${encodeURIComponent(code)}`);
      if (res.ok) { cache.i18n = await res.json(); applyTranslations(); }
    } catch (_) {}
  }
  function t(key, fallback) { return cache.i18n[key] || fallback || key; }
  function applyTranslations() {
    $$("[data-i18n]").forEach((node) => {
      const key = node.getAttribute("data-i18n");
      if (!cache.i18n[key]) return;
      node.textContent = cache.i18n[key];
    });
    $$("[data-i18n-title]").forEach((node) => {
      const key = node.getAttribute("data-i18n-title");
      if (!cache.i18n[key]) return;
      node.setAttribute("title", cache.i18n[key]);
    });
    document.documentElement.lang = els.langInput.value || "en";
  }

  // ===== Custom Apple dropdown =====
  function setLocaleValue(code, label) {
    els.langInput.value = code;
    els.langBtn.querySelector(".apple-dropdown-value").textContent = label;
    $$(".apple-dropdown-item", els.langMenu).forEach((it) => {
      it.classList.toggle("active", it.getAttribute("data-value") === code);
    });
    loadLocale(code);
  }
  function openDropdown() {
    els.langBtn.setAttribute("aria-expanded", "true");
    els.langMenu.classList.add("open");
    const active = $(".apple-dropdown-item.active", els.langMenu);
    if (active) active.focus();
  }
  function closeDropdown() {
    els.langBtn.setAttribute("aria-expanded", "false");
    els.langMenu.classList.remove("open");
    $(".apple-dropdown-item.focused", els.langMenu)?.classList.remove("focused");
  }
  function toggleDropdown() {
    if (els.langBtn.getAttribute("aria-expanded") === "true") closeDropdown();
    else openDropdown();
  }
  let focusedIdx = -1;
  function moveFocus(dir) {
    const items = $$(".apple-dropdown-item", els.langMenu);
    if (!items.length) return;
    items.forEach((it) => it.classList.remove("focused"));
    focusedIdx = (focusedIdx + dir + items.length) % items.length;
    items[focusedIdx].classList.add("focused");
    items[focusedIdx].scrollIntoView({ block: "nearest" });
  }
  els.langBtn.addEventListener("click", (e) => { e.stopPropagation(); toggleDropdown(); });
  els.langBtn.addEventListener("keydown", (e) => {
    if (e.key === "ArrowDown" || e.key === "ArrowUp") {
      e.preventDefault();
      if (els.langBtn.getAttribute("aria-expanded") !== "true") { openDropdown(); focusedIdx = 0; moveFocus(0); return; }
      moveFocus(e.key === "ArrowDown" ? 1 : -1);
    } else if (e.key === "Escape") { closeDropdown(); els.langBtn.focus(); }
    else if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      const focused = $(".apple-dropdown-item.focused", els.langMenu);
      if (focused) {
        const val = focused.getAttribute("data-value");
        setLocaleValue(val, focused.textContent.trim());
        closeDropdown();
      } else { toggleDropdown(); }
    }
  });
  $$(".apple-dropdown-item", els.langMenu).forEach((it, idx) => {
    it.addEventListener("click", () => {
      const val = it.getAttribute("data-value");
      setLocaleValue(val, it.textContent.trim());
      closeDropdown();
      els.langBtn.focus();
    });
    it.addEventListener("mouseenter", () => {
      $$(".apple-dropdown-item", els.langMenu).forEach((x) => x.classList.remove("focused"));
      it.classList.add("focused");
      focusedIdx = idx;
    });
  });
  document.addEventListener("click", (e) => {
    if (!els.langWrap.contains(e.target)) closeDropdown();
  });
  document.addEventListener("keydown", (e) => {
    if (e.key === "Escape") closeDropdown();
  });

  // ===== Theme slider =====
  const themePos = { dark: 0, auto: 1, light: 2 };
  function applyTheme(v) {
    document.documentElement.setAttribute("data-theme",
      v === "dark" ? "dark" : v === "light" ? "light" : "auto");
    els.themeBtns.forEach((b) => {
      b.classList.toggle("active", b.getAttribute("data-theme-set") === v);
    });
    if (els.sliderPill) {
      const pos = themePos[v] != null ? themePos[v] : 1;
      els.sliderPill.setAttribute("data-pos", String(pos));
    }
    try { localStorage.setItem("w2e.theme", v); } catch (_) {}
  }
  const savedTheme = (() => { try { return localStorage.getItem("w2e.theme") || "auto"; } catch (_) { return "auto"; } })();
  applyTheme(savedTheme);
  els.themeBtns.forEach((b) => b.addEventListener("click", () => applyTheme(b.getAttribute("data-theme-set"))));

  // ===== Target segmented =====
  function getActiveTarget() {
    const btn = els.targetGroup.querySelector(".seg-btn.active");
    return btn ? btn.getAttribute("data-target") : "windows";
  }
  function setActiveTarget(target) {
    $$("button", els.targetGroup).forEach((b) => {
      const on = b.getAttribute("data-target") === target;
      b.classList.toggle("active", on);
      b.setAttribute("aria-checked", String(on));
    });
  }
  els.targetGroup.addEventListener("click", (ev) => {
    const btn = ev.target.closest(".seg-btn");
    if (btn) setActiveTarget(btn.getAttribute("data-target"));
  });
  els.targetGroup.addEventListener("keydown", (ev) => {
    const btns = $$("button", els.targetGroup);
    const i = btns.findIndex((b) => b === document.activeElement);
    if (i < 0) return;
    let next = -1;
    if (ev.key === "ArrowRight" || ev.key === "ArrowDown") next = (i + 1) % btns.length;
    else if (ev.key === "ArrowLeft" || ev.key === "ArrowUp") next = (i - 1 + btns.length) % btns.length;
    else if (ev.key === "Home") next = 0;
    else if (ev.key === "End") next = btns.length - 1;
    if (next >= 0) { ev.preventDefault(); btns[next].focus(); setActiveTarget(btns[next].getAttribute("data-target")); }
  });

  // ===== iOS switch =====
  const switchEl = els.resizable;
  function setResizable(on) {
    switchEl.setAttribute("aria-checked", String(on));
    switchEl.classList.toggle("on", !!on);
  }
  setResizable(switchEl.getAttribute("aria-checked") === "true");
  switchEl.addEventListener("click", () => {
    const on = switchEl.getAttribute("aria-checked") !== "true";
    setResizable(on);
  });
  switchEl.addEventListener("keydown", (ev) => {
    if (ev.key === " " || ev.key === "Enter") {
      ev.preventDefault();
      const on = switchEl.getAttribute("aria-checked") !== "true";
      setResizable(on);
    }
  });

  // ===== Status / Report =====
  function setStatus(level, text) {
    els.status.hidden = false;
    els.status.style.display = "";
    els.status.className = `status ${level}`;
    els.status.textContent = text;
  }
  function cssClassFor(s) {
    if (s === "ok") return "ok";
    if (s === "warn") return "warn";
    if (s === "fail") return "fail";
    if (s === "error") return "fail";
    return "";
  }
  function escapeHtml(s) {
    return String(s == null ? "" : s)
      .replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
  }
  function setReport(items) {
    if (!items || items.length === 0) { hideEl(els.report); els.report.innerHTML = ""; return; }
    showEl(els.report);
    els.report.innerHTML = items
      .map((it) => `<li class="${cssClassFor(it.status || it.Level)}">${escapeHtml(it.check || it.Check || "")} — ${escapeHtml(it.message || it.Message || "")}</li>`)
      .join("");
  }

  // ===== Wizard navigation =====
  function updateIndicator() {
    els.dots.forEach((dot, i) => {
      const step = i + 1;
      dot.classList.toggle("active", step === currentStep);
      dot.classList.toggle("done", step < currentStep);
    });
    els.lines.forEach((line, i) => {
      line.classList.toggle("filled", i < currentStep - 1);
    });
    els.indicator.setAttribute("aria-valuenow", currentStep);

    // ——— Button visibility: use style.display for reliability ———
    // Step 1: hide prev
    if (currentStep <= 1) { hideEl(els.prevBtn); } else { showEl(els.prevBtn); }
    // Step 4: hide next
    if (currentStep >= TOTAL_STEPS) { hideEl(els.nextBtn); } else { showEl(els.nextBtn); }
    // Validate + Start: only on step 4
    if (currentStep === TOTAL_STEPS) { showEl(els.validate); } else { hideEl(els.validate); }
    // Start: only on step 4 AND only after validation passed
    if (currentStep === TOTAL_STEPS && validated) { showEl(els.start); } else { hideEl(els.start); }
  }

  function goToStep(n) {
    if (n < 1 || n > TOTAL_STEPS) return;
    // Hide current
    els.steps.forEach((s) => s.classList.remove("active"));
    currentStep = n;
    // Show new
    const target = $(`.wizard-step[data-step="${currentStep}"]`);
    if (target) target.classList.add("active");
    updateIndicator();
    // Clear status when switching steps
    hideEl(els.status);
    hideEl(els.report);
    // Focus first input on new step
    const firstInput = $(".glass-input", target);
    if (firstInput) setTimeout(() => firstInput.focus(), 100);
  }

  function canAdvance() {
    // Step-level validation
    if (currentStep === 2) {
      if (!els.sourceDir.value.trim()) {
        setStatus("warn", t("status.needSource", "Please select a web source directory."));
        return false;
      }
    }
    if (currentStep === 3) {
      if (!els.output.value.trim()) {
        setStatus("warn", t("status.needOutput", "Please specify an output path."));
        return false;
      }
    }
    return true;
  }

  els.nextBtn.addEventListener("click", () => {
    if (canAdvance()) goToStep(currentStep + 1);
  });
  els.prevBtn.addEventListener("click", () => {
    goToStep(currentStep - 1);
  });

  // Keyboard: Enter to advance (except in textarea / button contexts)
  document.addEventListener("keydown", (e) => {
    if (e.key === "Enter" && !e.target.closest("textarea, button, .apple-dropdown-trigger")) {
      e.preventDefault();
      if (currentStep < TOTAL_STEPS && els.nextBtn.style.display !== "none") {
        els.nextBtn.click();
      } else if (currentStep === TOTAL_STEPS && els.start.style.display !== "none") {
        els.start.click();
      }
    }
  });

  // ===== Build orchestration =====
  async function postJSON(url, body) {
    const res = await fetch(url, {
      method: "POST", headers: { "Content-Type": "application/json; charset=utf-8" },
      body: JSON.stringify(body),
    });
    return res.json();
  }
  async function getJSON(url) { const res = await fetch(url); return res.json(); }

  function configFromForm() {
    return {
      source_dir: els.sourceDir.value.trim(),
      output: els.output.value.trim(),
      window_title: els.title.value.trim(),
      width: parseInt(els.width.value, 10) || 1024,
      height: parseInt(els.height.value, 10) || 720,
      resizable: els.resizable.getAttribute("aria-checked") === "true",
      icon: els.icon.value.trim(),
      target: getActiveTarget(),
    };
  }
  function busy(yes) {
    els.start.disabled = !!yes;
    els.validate.disabled = !!yes;
    els.nextBtn.disabled = !!yes;
    els.prevBtn.disabled = !!yes;
  }

  // Validate
  els.validate.addEventListener("click", async () => {
    const cfg = configFromForm();
    if (!cfg.source_dir) { setStatus("warn", t("status.needSource", "Provide a source directory first.")); return; }
    busy(true); setStatus("warn", t("status.validating", "Validating…"));
    try {
      const rep = await postJSON("/api/validate", { source_dir: cfg.source_dir });
      if (rep.results && rep.results.length) { setReport(rep.results); }
      validated = !!rep.ready;
      setStatus(rep.ready ? "ok" : "warn",
        rep.ready ? t("status.ready", "Project is ready for build.")
                  : t("status.notReady", "Validation produced warnings — see details."));
      updateIndicator(); // refresh start button visibility
    } catch (e) {
      validated = false;
      setStatus("fail", String(e));
    } finally { busy(false); }
  });

  // Build
  els.start.addEventListener("click", async () => {
    if (!validated) {
      setStatus("warn", t("status.validateFirst", "Please validate first before building."));
      return;
    }
    const cfg = configFromForm();
    if (!cfg.source_dir) { setStatus("warn", t("status.needSource", "Provide a source directory first.")); return; }
    if (!cfg.output) { setStatus("warn", t("status.needOutput", "Specify an output path.")); return; }
    busy(true); setReport([]); setStatus("warn", t("status.building", "Packaging…"));
    try {
      const res = await postJSON("/api/build", cfg);
      renderBuildResult(res, cfg.target);
    } catch (e) {
      setStatus("fail", String(e));
    } finally { busy(false); }
  });

  function renderBuildResult(res, requestedTarget) {
    const isMulti = Array.isArray(res.targets) || Array.isArray(res.Targets);
    if (isMulti) {
      const targets = res.targets || res.Targets || [];
      const allOk = (res.success == null ? res.Success : res.success) === true ||
                    targets.every((t) => (t.success || t.Success) === true);
      const lines = targets.map((t) => {
        const ok = (t.success == null ? t.Success : t.success) === true;
        const path = t.output_path || t.OutputPath || t.Path || "";
        const sizeMB = ((t.size || t.Size || 0) / 1024 / 1024).toFixed(2);
        const fmt = t.format || t.Format || "";
        const err = t.error_message || t.ErrorMessage || t.error || "";
        return {
          status: ok ? "ok" : "fail",
          check: (t.target || t.Target || "").toString().toUpperCase(),
          message: ok ? `${path} • ${sizeMB} MB • ${fmt}` : err || t.error_code || t.ErrorCode || "failed",
        };
      });
      setReport(lines);
      if (allOk) {
        setStatus("ok", t("status.targetAllDone", "All requested binaries produced."));
      } else if (targets.some((x) => (x.success == null ? x.Success : x.success) === true)) {
        setStatus("warn", t("status.targetPartial", "Some targets failed — see details."));
      } else {
        const firstFail = targets[0] || {};
        const code = firstFail.error_code || firstFail.ErrorCode || "";
        const msg = firstFail.error_message || firstFail.ErrorMessage || "unknown error";
        setStatus("fail", `${t("status.failed", "Build failed")}${code ? ` [${code}]` : ""}: ${msg}`);
      }
      return;
    }
    // Single target
    if ((res.success == null ? res.Success : res.success) === true) {
      const sizeMB = ((res.size || res.Size || 0) / 1024 / 1024).toFixed(2);
      setStatus("ok", `${t("status.done", "Done")} — ${res.output_path || res.OutputPath || res.path || ""} (${sizeMB} MB)`);
      if (res.warnings && res.warnings.length) {
        setReport(res.warnings.map((w) => ({ status: "warn", check: "warning", message: w })));
      }
    } else {
      const code = res.error_code || res.ErrorCode ? `${res.error_code || res.ErrorCode}` : "";
      const msg = res.error_message || res.ErrorMessage || res.error || "unknown error";
      setStatus("fail", `${t("status.failed", "Build failed")}${code ? ` [${code}]` : ""}: ${msg}`);
    }
  }

  // ===== Folder / file pickers =====
  let _pickerBusy = false;
  function _pickerSafe() { _pickerBusy = false; }

  async function _openPicker(targetInput, isFile) {
    if (_pickerBusy) return;
    _pickerBusy = true;
    const timer = setTimeout(_pickerSafe, 10000);
    try {
      let picked = null;
      if (isFile && typeof window.__pickFile === "function") {
        picked = await window.__pickFile();
      } else if (!isFile && typeof window.__pickDirectory === "function") {
        picked = await window.__pickDirectory();
      } else {
        const api = isFile ? "/api/pickFile" : "/api/pickDirectory";
        const res = await fetch(api, { method: "POST" });
        const data = await res.json();
        picked = data && data.path ? data.path : null;
      }
      if (picked) {
        targetInput.value = picked;
        targetInput.dispatchEvent(new Event("input"));
        w2eLog.report("pickOK", picked);
      }
    } catch (e) { w2eLog.report("pickErr", e); }
    clearTimeout(timer);
    _pickerBusy = false;
  }

  // Bind browse buttons — each button has a data-browse attribute targeting its input
  $$('[data-browse]').forEach((btn) => {
    btn.addEventListener("click", () => {
      const targetId = btn.getAttribute("data-browse");
      const input = document.getElementById(targetId);
      if (!input) return;
      const isFile = targetId === "icon";
      _openPicker(input, isFile);
    });
  });

  // ===== Auto-populate output path =====
  function _suggestOutput() {
    const src = els.sourceDir.value.trim();
    if (!src) return;
    // Only auto-fill if output is empty
    if (els.output.value.trim()) return;
    const title = els.title.value.trim();
    const baseName = title || src.replace(/[\\/]+$/, "").split(/[\\/]/).pop() || "MyApp";
    // Output: same directory as source, filename = sanitized title.exe
    const safeName = baseName.replace(/[<>:"/\\|?*]/g, "_");
    els.output.value = src + "\\" + safeName + ".exe";
  }
  // Watch source dir changes
  els.sourceDir.addEventListener("input", _suggestOutput);
  els.sourceDir.addEventListener("change", _suggestOutput);
  els.sourceDir.addEventListener("blur", _suggestOutput);
  // Also watch title changes — if output is still the auto-suggested one, update it
  let _lastAutoOutput = "";
  function _watchOutputUpdate() {
    const src = els.sourceDir.value.trim();
    if (!src) return;
    const title = els.title.value.trim();
    if (!title) return;
    const safeName = title.replace(/[<>:"/\\|?*]/g, "_");
    const autoVal = src + "\\" + safeName + ".exe";
    // Only update if output is empty or was previously auto-generated
    if (!els.output.value.trim() || els.output.value.trim() === _lastAutoOutput) {
      els.output.value = autoVal;
      _lastAutoOutput = autoVal;
    }
  }
  els.title.addEventListener("input", _watchOutputUpdate);

  // ===== Initial state: hide all nav buttons, then show correct ones =====
  hideEl(els.prevBtn);
  hideEl(els.nextBtn);
  hideEl(els.validate);
  hideEl(els.start);
  hideEl(els.status);
  hideEl(els.report);
  showEl(els.nextBtn);

  updateIndicator();
  getJSON("/api/state").then((st) => {
    if (st.available_locales && Array.isArray(st.available_locales)) {
      const code = st.current_locale || "en";
      const item = $(".apple-dropdown-item[data-value='" + code + "']", els.langMenu);
      const label = item ? item.textContent.trim() : code;
      setLocaleValue(code, label);
    } else {
      setLocaleValue("en", "English");
    }
    if (st.default_width) els.width.value = st.default_width;
    if (st.default_height) els.height.value = st.default_height;
    if (st.min_width) els.width.min = st.min_width;
    if (st.min_height) els.height.min = st.min_height;
    if (st.version) els.appVersion.textContent = st.version;
  }).catch(() => { loadLocale("en"); });
})();
