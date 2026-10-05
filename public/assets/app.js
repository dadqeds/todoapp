// Относительный путь: UI и API отдаются одним сервером, CORS не нужен.
const API_BASE = "/api/v1";
// Telegram Mini App. Вне Telegram (локальный адрес) объект есть, но initData пустой.
const tg = window.Telegram?.WebApp;
const inTelegram = Boolean(tg?.initData);
const PAGE_SIZE = 50;
const LIST_COLORS = ["green", "violet", "coral", "blue", "pink", "amber"];
const COLOR_NAMES = {
  green: "Зелёный",
  violet: "Фиолетовый",
  coral: "Коралловый",
  blue: "Синий",
  pink: "Розовый",
  amber: "Янтарный",
};

const state = {
  tab: "tasks",
  me: null,
  lists: [],
  listFilter: "all",
  tasks: [],
  hasMore: false,
  statsPeriod: "week",
  statsUser: "",
  users: [],
};

const $ = s => document.querySelector(s);
const esc = v =>
  String(v ?? "").replace(
    /[&<>'"]/g,
    c => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", "'": "&#039;", '"': "&quot;" })[c],
  );
const isAdmin = () => Boolean(state.me?.is_admin);

const ICONS = {
  tasks: '<path d="M9 6h11M9 12h11M9 18h11"/><path d="M4 6l1 1 2-2M4 12l1 1 2-2M4 18l1 1 2-2"/>',
  stats: '<path d="M5 20V11M12 20V4M19 20v-6"/>',
  profile: '<circle cx="12" cy="8" r="4"/><path d="M4 21c1-4 4-6 8-6s7 2 8 6"/>',
  plus: '<path d="M12 5v14M5 12h14"/>',
  check: '<path d="M5 12l4 4L19 6"/>',
  close: '<path d="M6 6l12 12M18 6L6 18"/>',
  chev: '<path d="M9 6l6 6-6 6"/>',
  back: '<path d="M15 6l-6 6 6 6"/>',
  edit: '<path d="M4 20h4L19 9l-4-4L4 16v4z"/>',
  bell: '<path d="M6 16V11a6 6 0 0112 0v5l2 2H4l2-2z"/><path d="M10 20a2 2 0 004 0"/>',
  repeat: '<path d="M4 11V9a3 3 0 013-3h12M16 3l3 3-3 3"/><path d="M20 13v2a3 3 0 01-3 3H5M8 21l-3-3 3-3"/>',
  share: '<path d="M12 4v11M8 8l4-4 4 4"/><path d="M5 13v5a2 2 0 002 2h10a2 2 0 002-2v-5"/>',
  copy: '<rect x="8" y="8" width="12" height="12" rx="2"/><path d="M16 8V6a2 2 0 00-2-2H6a2 2 0 00-2 2v8a2 2 0 002 2h2"/>',
  users: '<circle cx="9" cy="8" r="3"/><path d="M3 20c.8-3.8 3-6 6-6s5.2 2.2 6 6"/><path d="M15 5.5a3 3 0 010 5.9M17 14c2.2.6 3.5 2.2 4 5"/>',
};
const icon = (name, size = 20) =>
  `<svg width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${ICONS[name] || ""}</svg>`;
const colorStyle = color => `style="--list-color: var(--c-${LIST_COLORS.includes(color) ? color : "blue"})"`;

// ---------- Склонения и даты ----------
function plural(n, one, few, many) {
  const m10 = n % 10,
    m100 = n % 100;
  if (m10 === 1 && m100 !== 11) return one;
  if (m10 >= 2 && m10 <= 4 && (m100 < 12 || m100 > 14)) return few;
  return many;
}
const startOfDay = d => new Date(d.getFullYear(), d.getMonth(), d.getDate());
const dayDiff = (a, b) => Math.round((startOfDay(a) - startOfDay(b)) / 86400000);
const pad = n => String(n).padStart(2, "0");
const toDateInput = d => `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
const toTimeInput = d => `${pad(d.getHours())}:${pad(d.getMinutes())}`;
const fmtTime = d => d.toLocaleTimeString("ru-RU", { hour: "2-digit", minute: "2-digit" });

function fmtDay(d, now = new Date()) {
  const diff = dayDiff(d, now);
  if (diff === 0) return "сегодня";
  if (diff === 1) return "завтра";
  if (diff === -1) return "вчера";
  const opts = { day: "numeric", month: "short" };
  if (d.getFullYear() !== now.getFullYear()) opts.year = "numeric";
  return d.toLocaleDateString("ru-RU", opts).replace(" г.", "").replace(".", "");
}

// Метка срока: красная — просрочено, жёлтая — сегодня, серая — позже.
function dueTag(task) {
  if (!task.due_at) return "";
  const due = new Date(task.due_at);
  const now = new Date();
  const text = task.due_all_day ? fmtDay(due, now) : `${fmtDay(due, now)}, ${fmtTime(due)}`;
  if (task.completed) return `<span class="tag">${esc(text)}</span>`;
  const overdue = task.due_all_day ? dayDiff(due, now) < 0 : due < now;
  if (overdue) return `<span class="tag over">просрочено, ${esc(text)}</span>`;
  if (dayDiff(due, now) === 0) return `<span class="tag today">${esc(text)}</span>`;
  return `<span class="tag">${esc(text)}</span>`;
}

const REMIND_OPTIONS = [
  ["none", "Нет"],
  ["0", "В срок"],
  ["15", "За 15 мин"],
  ["60", "За час"],
  ["1440", "За день"],
];
const WEEKDAYS = ["пн", "вт", "ср", "чт", "пт", "сб", "вс"];
const isoWeekday = d => ((d.getDay() + 6) % 7) + 1;

function repeatText(r) {
  if (!r) return "";
  if (r.kind === "daily" || (r.kind === "weekly" && r.weekdays?.length === 7)) return "каждый день";
  if (r.kind === "weekly") return r.weekdays.map(d => WEEKDAYS[d - 1]).join(", ");
  if (r.kind === "monthly") return "каждый месяц";
  return "каждый год";
}
const sameRepeat = (a, b) =>
  (!a && !b) || (a && b && a.kind === b.kind && (a.weekdays || []).join() === (b.weekdays || []).join());

// Срок «на весь день» хранится как конец дня по времени пользователя.
function buildDue(date, time) {
  if (!date) return { due_at: null, due_all_day: false };
  const [y, m, d] = date.split("-").map(Number);
  if (!time) return { due_at: new Date(y, m - 1, d, 23, 59, 59).toISOString(), due_all_day: true };
  const [hh, mm] = time.split(":").map(Number);
  return { due_at: new Date(y, m - 1, d, hh, mm).toISOString(), due_all_day: false };
}

function fmtDuration(seconds) {
  if (seconds == null) return "—";
  if (seconds < 60) return "меньше минуты";
  const min = Math.round(seconds / 60);
  if (min < 60) return `${min} мин`;
  const h = Math.floor(min / 60),
    restMin = min % 60;
  if (h < 24) return restMin ? `${h} ч ${restMin} мин` : `${h} ч`;
  const d = Math.floor(h / 24),
    restH = h % 24;
  const days = `${d} ${plural(d, "день", "дня", "дней")}`;
  return restH ? `${days} ${restH} ч` : days;
}

const initials = name =>
  String(name || "?")
    .trim()
    .split(/\s+/)
    .slice(0, 2)
    .map(x => x[0])
    .join("")
    .toUpperCase();

// ---------- Уведомления ----------
function toast(message, error = false) {
  const t = document.createElement("div");
  t.className = "toast" + (error ? " error" : "");
  t.setAttribute("role", error ? "alert" : "status");
  t.textContent = message;
  $("#toasts").appendChild(t);
  setTimeout(() => t.remove(), 3200);
}
const haptic = type => tg?.HapticFeedback?.notificationOccurred?.(type);

// ---------- API ----------
// Понятные сообщения по коду ответа: сервер отдаёт технический текст на английском.
const DEFAULT_MESSAGES = {
  400: "Проверьте введённые данные.",
  401: "Откройте приложение через Telegram-бота.",
  403: "Недостаточно прав.",
  404: "Запись не найдена — возможно, её уже удалили.",
  409: "Данные изменились в другом окне. Список обновлён, повторите действие.",
};
class ApiError extends Error {
  constructor(status, message) {
    super(message);
    this.status = status;
  }
}
async function api(path, options = {}, messages = {}) {
  let r;
  try {
    r = await fetch(API_BASE + path, {
      ...options,
      headers: {
        "Content-Type": "application/json",
        // Сервер проверяет подпись initData токеном бота.
        ...(inTelegram ? { Authorization: `tma ${tg.initData}` } : {}),
        ...(options.headers || {}),
      },
    });
  } catch {
    throw new ApiError(0, "Нет связи с сервером. Проверьте интернет и попробуйте ещё раз.");
  }
  const text = await r.text();
  let data = null;
  if (text) {
    try {
      data = JSON.parse(text);
    } catch {
      data = text;
    }
  }
  if (!r.ok) {
    const message =
      messages[r.status] ||
      DEFAULT_MESSAGES[r.status] ||
      (r.status >= 500 ? "Ошибка сервера. Попробуйте позже." : `Ошибка запроса (${r.status}).`);
    throw new ApiError(r.status, message);
  }
  return data;
}
const send = (method, body) => ({ method, body: body === undefined ? undefined : JSON.stringify(body) });

// ---------- Данные ----------
async function loadLists() {
  const lists = await api("/lists");
  state.lists = Array.isArray(lists) ? lists : [];
}
const listById = id => state.lists.find(l => l.id === id);
const isShared = list => (list?.members?.length || 0) > 1;
const isOwner = list => list?.role !== "member";
const memberName = (list, userId) => list?.members?.find(m => m.user_id === userId)?.full_name;
const defaultList = () => state.lists.find(l => l.is_default) || state.lists[0];

// ---------- Каркас ----------
const TABS = { tasks: "Задачи", stats: "Статистика", profile: "Профиль" };
function renderTabbar() {
  document.querySelectorAll("#tabbar button").forEach(b => {
    const tab = b.dataset.tab;
    b.innerHTML = `${icon(tab, 22)}<span>${TABS[tab]}</span>`;
    const active = tab === state.tab || (tab === "profile" && state.tab === "users");
    if (active) b.setAttribute("aria-current", "page");
    else b.removeAttribute("aria-current");
    b.onclick = () => switchTab(tab);
  });
}

// Кнопка «Добавить задачу»: в Telegram — системная, иначе своя над меню.
function updateAddButton() {
  const visible = state.tab === "tasks" && !isSheetOpen();
  if (inTelegram) {
    $("#actionBar").hidden = true;
    if (visible) tg.MainButton.show();
    else tg.MainButton.hide();
    return;
  }
  $("#actionBar").hidden = !visible;
}

function renderLoader() {
  $("#screen").innerHTML = '<div class="loader" role="status" aria-label="Загрузка"></div>';
}

function renderError(e, retry) {
  $("#screen").innerHTML = `<div class="empty"><h2>Не получилось загрузить</h2><p>${esc(e.message)}</p>${
    retry ? '<button class="btn primary" id="retryBtn" type="button">Повторить</button>' : ""
  }</div>`;
  if (retry) $("#retryBtn").onclick = retry;
}

async function switchTab(tab) {
  if (isSheetOpen()) closeSheet(true);
  state.tab = tab;
  renderTabbar();
  updateAddButton();
  updateBackButton();
  window.scrollTo(0, 0);
  if (tab === "tasks") return loadTasks();
  if (tab === "stats") return loadStats();
  if (tab === "users") return loadUsers();
  return renderProfile();
}

// ---------- Задачи ----------
async function loadTasks({ append = false } = {}) {
  if (!append) renderLoader();
  try {
    if (!state.lists.length) await loadLists();
    // Без user_id сервер отдаёт задачи из своих и общих списков.
    const params = new URLSearchParams({
      limit: PAGE_SIZE + 1,
      offset: append ? state.tasks.length : 0,
    });
    if (state.listFilter !== "all") params.set("list_id", state.listFilter);
    const page = (await api(`/tasks?${params}`)) || [];
    state.hasMore = page.length > PAGE_SIZE;
    const items = page.slice(0, PAGE_SIZE);
    state.tasks = append ? state.tasks.concat(items) : items;
    renderTasks();
  } catch (e) {
    renderError(e, () => loadTasks());
  }
}

function listChips() {
  const total = state.lists.reduce((sum, l) => sum + l.open_tasks, 0);
  const chip = (value, label, count, color, shared) =>
    `<button class="chip" type="button" data-list="${value}" aria-pressed="${String(state.listFilter) === String(value)}" ${
      color ? colorStyle(color) : ""
    }>${color ? '<span class="dot"></span>' : ""}${esc(label)}${
      shared ? `<span class="count" aria-label="общий">${icon("users", 14)}</span>` : ""
    }<span class="count">${count}</span></button>`;
  return `<div class="chips" role="group" aria-label="Списки">${chip("all", "Все", total)}${state.lists
    .map(l => chip(l.id, l.title, l.open_tasks, l.color, isShared(l)))
    .join("")}<button class="chip" type="button" id="newListChip" aria-label="Новый список">${icon("plus", 16)}</button></div>`;
}

function taskCard(t) {
  const list = listById(t.list_id);
  const showDot = state.listFilter === "all" && list;
  return `<article class="task${t.completed ? " done" : ""}" data-task="${t.id}">
    <button class="task-check" type="button" data-check="${t.id}" aria-pressed="${t.completed}" aria-label="${
      t.completed ? "Вернуть в работу" : "Отметить выполненной"
    }: ${esc(t.title)}">${t.completed ? icon("check", 14) : ""}</button>
    <div class="task-body">
      <div class="task-title">${esc(t.title)}</div>
      ${t.description ? `<div class="task-desc">${esc(t.description)}</div>` : ""}
      <div class="task-meta">${dueTag(t)}${
        t.repeat && !t.completed ? `<span class="tag">${icon("repeat", 12)} ${esc(repeatText(t.repeat))}</span>` : ""
      }${
        t.remind_before_minutes != null && !t.completed
          ? `<span class="tag" title="Напоминание">${icon("bell", 12)}</span>`
          : ""
      }${
        isShared(list) && t.author_user_id !== state.me.id
          ? `<span class="tag">${esc(memberName(list, t.author_user_id) || "участник")}</span>`
          : ""
      }</div>
      ${checklistProgress(t, list)}
    </div>
    ${showDot ? `<span class="dot" ${colorStyle(list.color)} title="${esc(list.title)}"></span>` : ""}
  </article>`;
}

// Прогресс чеклиста на карточке: «3/7» и полоска цвета списка.
function checklistProgress(t, list) {
  if (!t.items_total) return "";
  const pct = Math.round((t.items_done / t.items_total) * 100);
  return `<div class="progress" ${list ? colorStyle(list.color) : ""} aria-label="Пункты: ${t.items_done} из ${t.items_total}">
    <div class="progress-track"><div class="progress-fill" style="width:${pct}%"></div></div>
    <span class="progress-count">${t.items_done}/${t.items_total}</span></div>`;
}

function renderTasks() {
  const filtered = state.listFilter !== "all";
  const emptyText = filtered
    ? "<h2>В этом списке пусто</h2><p>Добавьте первую задачу или выберите другой список.</p>"
    : "<h2>Задач пока нет</h2><p>Запишите первое дело — оно появится здесь.</p>";
  $("#screen").innerHTML = `
    <div class="screen-head"><h1 class="title">Мои задачи</h1></div>
    ${listChips()}
    ${
      state.tasks.length
        ? `<div class="tasks">${state.tasks.map(taskCard).join("")}</div>`
        : `<div class="empty">${emptyText}</div>`
    }
    ${state.hasMore ? '<button class="btn quiet block more" type="button" id="moreBtn">Показать ещё</button>' : ""}`;

  document.querySelectorAll("[data-list]").forEach(b => {
    b.onclick = () => {
      state.listFilter = b.dataset.list === "all" ? "all" : Number(b.dataset.list);
      loadTasks();
    };
  });
  $("#newListChip").onclick = () => openListSheet();
  if (state.hasMore) $("#moreBtn").onclick = () => loadTasks({ append: true });
  document.querySelectorAll("[data-task]").forEach(card => {
    card.onclick = e => {
      if (e.target.closest("[data-check]")) return;
      openTaskSheet(state.tasks.find(t => t.id === Number(card.dataset.task)));
    };
  });
  document.querySelectorAll("[data-check]").forEach(b => {
    b.onclick = () => toggleTask(Number(b.dataset.check));
  });
}

// На 409/404 (изменили в другом окне) перезагружаем, чтобы показать актуальное.
async function handleError(err, reload) {
  toast(err.message, true);
  if (err.status === 409 || err.status === 404) await reload();
}

async function refreshListsAndTasks() {
  await loadLists();
  await loadTasks();
}

async function toggleTask(id) {
  const t = state.tasks.find(x => x.id === id);
  if (!t) return;
  try {
    await api(`/tasks/${id}`, send("PATCH", { completed: !t.completed, version: t.version }));
    if (!t.completed) haptic("success");
    toast(t.completed ? "Задача снова в работе" : t.repeat ? "Готово. Следующая уже в списке" : "Задача выполнена");
    await refreshListsAndTasks();
  } catch (err) {
    handleError(err, refreshListsAndTasks);
  }
}

// ---------- Нижний лист ----------
let sheetDirty = false;
let sheetOnSubmit = null;
let lastFocused = null;
const isSheetOpen = () => $("#sheetBackdrop").classList.contains("open");

function openSheet(title, body, { onSubmit } = {}) {
  lastFocused = document.activeElement;
  sheetDirty = false;
  sheetOnSubmit = onSubmit || null;
  $("#sheet").innerHTML = `<div class="sheet-grip"></div>
    <div class="sheet-head"><h2 class="sheet-title" id="sheetTitle">${title}</h2>
    <button class="icon-btn" type="button" id="sheetClose" aria-label="Закрыть">${icon("close")}</button></div>
    ${body}`;
  $("#sheetBackdrop").classList.add("open");
  document.body.style.overflow = "hidden";
  $("#sheetClose").onclick = () => closeSheet();
  const form = $("#sheet form");
  if (form) {
    form.addEventListener("input", () => setDirty(true));
    form.addEventListener("submit", e => {
      e.preventDefault();
      sheetOnSubmit?.();
    });
  }
  updateAddButton();
  updateBackButton();
  $("#sheet").querySelector("input, textarea, select, button.chip")?.focus({ preventScroll: true });
}

function setDirty(value) {
  sheetDirty = value;
  // В Telegram спрашиваем подтверждение, если окно закрывают с несохранённой формой.
  if (inTelegram) value ? tg.enableClosingConfirmation() : tg.disableClosingConfirmation();
}

async function closeSheet(force = false) {
  if (!isSheetOpen()) return;
  if (!force && sheetDirty && !(await ask("Закрыть без сохранения?"))) return;
  setDirty(false);
  // Пункты сохраняются сразу, поэтому после листа с изменённым чеклистом
  // обновляем карточки: на них счётчик «3/7».
  const checklistChanged = checklist?.changed;
  checklist = null;
  if (checklistChanged) loadTasks();
  $("#sheetBackdrop").classList.remove("open");
  document.body.style.overflow = "";
  updateAddButton();
  updateBackButton();
  if (lastFocused && document.contains(lastFocused)) lastFocused.focus({ preventScroll: true });
}

function updateBackButton() {
  if (!inTelegram) return;
  if (isSheetOpen() || state.tab === "users") tg.BackButton.show();
  else tg.BackButton.hide();
}

function ask(message) {
  if (inTelegram && tg.showConfirm) return new Promise(resolve => tg.showConfirm(message, resolve));
  return Promise.resolve(confirm(message));
}

$("#sheetBackdrop").addEventListener("click", e => {
  if (e.target.id === "sheetBackdrop") closeSheet();
});
document.addEventListener("keydown", e => {
  if (!isSheetOpen()) return;
  if (e.key === "Escape") return closeSheet();
  // Не выпускаем фокус из открытого листа.
  if (e.key === "Tab") {
    const focusable = [...$("#sheet").querySelectorAll("button, input, select, textarea")].filter(
      el => !el.disabled && el.offsetParent !== null,
    );
    if (!focusable.length) return;
    const first = focusable[0],
      last = focusable[focusable.length - 1];
    if (e.shiftKey && document.activeElement === first) {
      e.preventDefault();
      last.focus();
    } else if (!e.shiftKey && document.activeElement === last) {
      e.preventDefault();
      first.focus();
    }
  }
});

// Группа чипов с одним выбранным значением.
function bindChoice(selector, onChange) {
  const buttons = [...document.querySelectorAll(selector)];
  buttons.forEach(b => {
    b.onclick = () => {
      buttons.forEach(x => x.setAttribute("aria-pressed", String(x === b)));
      setDirty(true);
      onChange?.(b.dataset.value);
    };
  });
}
const chosen = selector => document.querySelector(`${selector}[aria-pressed="true"]`)?.dataset.value;

// ---------- Форма задачи ----------
function openTaskSheet(task = null) {
  const now = new Date();
  const due = task?.due_at ? new Date(task.due_at) : null;
  const today = toDateInput(now);
  const tomorrow = toDateInput(new Date(now.getFullYear(), now.getMonth(), now.getDate() + 1));
  const dueDate = due ? toDateInput(due) : "";
  const preset = !due ? "none" : dueDate === today ? "today" : dueDate === tomorrow ? "tomorrow" : "date";
  const listId = task?.list_id ?? (state.listFilter !== "all" ? state.listFilter : defaultList()?.id);
  const dueChip = (value, label) =>
    `<button class="chip" type="button" data-due="" data-value="${value}" aria-pressed="${preset === value}">${label}</button>`;

  openSheet(
    task ? "Задача" : "Новая задача",
    `<form id="taskForm" novalidate>
      <input class="field" id="taskTitle" maxlength="100" placeholder="Что нужно сделать" value="${esc(task?.title)}" aria-label="Название" />
      <textarea class="field" id="taskDesc" maxlength="1000" placeholder="Подробности, если нужны" aria-label="Описание" style="margin-top:8px">${esc(
        task?.description,
      )}</textarea>

      <span class="field-label">Чеклист</span>
      <div class="checklist" id="checklist"></div>

      <span class="field-label">Срок</span>
      <div class="chips wrap">${dueChip("none", "Без срока")}${dueChip("today", "Сегодня")}${dueChip(
        "tomorrow",
        "Завтра",
      )}${dueChip("date", "Другая дата")}</div>
      <div id="dueFields" ${preset === "none" ? "hidden" : ""} style="margin-top:8px">
        <div class="inline">
          <input class="field" type="date" id="taskDate" value="${dueDate || today}" aria-label="Дата" ${
            preset === "date" ? "" : "hidden"
          } />
          <input class="field" type="time" id="taskTime" value="${due && !task.due_all_day ? toTimeInput(due) : ""}" aria-label="Время" />
        </div>
        <p class="hint">Без времени — срок до конца дня.</p>

        <span class="field-label">Повтор</span>
        <div class="chips wrap">${[
          ["none", "Нет"],
          ["daily", "Каждый день"],
          ["weekly", "По дням недели"],
          ["monthly", "Каждый месяц"],
          ["yearly", "Каждый год"],
        ]
          .map(
            ([v, label]) =>
              `<button class="chip" type="button" data-repeat="" data-value="${v}" aria-pressed="${(task?.repeat?.kind || "none") === v}">${label}</button>`,
          )
          .join("")}</div>
        <div class="chips wrap" id="weekdayPicker" style="margin-top:8px" ${task?.repeat?.kind === "weekly" ? "" : "hidden"} role="group" aria-label="Дни недели">${WEEKDAYS.map(
          (d, i) =>
            `<button class="chip weekday" type="button" data-weekday="${i + 1}" aria-pressed="${Boolean(
              task?.repeat?.weekdays?.includes(i + 1),
            )}">${d}</button>`,
        ).join("")}</div>

        <span class="field-label">Напомнить</span>
        <div class="chips wrap">${REMIND_OPTIONS.map(
          ([v, label]) =>
            `<button class="chip" type="button" data-remind="" data-value="${v}" aria-pressed="${
              String(task?.remind_before_minutes ?? "none") === v
            }" ${v === "15" || v === "60" ? 'data-timed=""' : ""}>${label}</button>`,
        ).join("")}</div>
        ${
          state.me.remind_enabled
            ? ""
            : '<p class="hint">Напоминания выключены — включите их в профиле, чтобы бот писал.</p>'
        }
      </div>

      <span class="field-label">Список</span>
      <div class="chips wrap">${state.lists
        .map(
          l =>
            `<button class="chip" type="button" data-tlist="" data-value="${l.id}" aria-pressed="${l.id === listId}" ${colorStyle(
              l.color,
            )}><span class="dot"></span>${esc(l.title)}</button>`,
        )
        .join("")}</div>

      <div class="sheet-actions">
        <button class="btn primary block" type="submit">${task ? "Сохранить" : "Создать задачу"}</button>
        ${
          task
            ? `<button class="btn quiet block" type="button" id="taskToggle">${
                task.completed ? "Вернуть в работу" : "Отметить выполненной"
              }</button><button class="btn danger block" type="button" id="taskDelete">Удалить задачу</button>`
            : ""
        }
      </div>
    </form>`,
    { onSubmit: () => saveTask(task) },
  );

  bindChoice("[data-due]", value => {
    $("#dueFields").hidden = value === "none";
    $("#taskDate").hidden = value !== "date";
  });
  bindChoice("[data-repeat]", value => {
    $("#weekdayPicker").hidden = value !== "weekly";
    // По умолчанию — день недели выбранного срока.
    if (value === "weekly" && !document.querySelector('[data-weekday][aria-pressed="true"]')) {
      const due = readDue().due_at;
      const day = isoWeekday(due ? new Date(due) : new Date());
      document.querySelector(`[data-weekday="${day}"]`).setAttribute("aria-pressed", "true");
    }
  });
  // Для задачи на весь день «за 15 минут» и «за час» смысла не имеют.
  let remindTouched = Boolean(task);
  bindChoice("[data-remind]", () => (remindTouched = true));
  const syncRemindOptions = () => {
    const timed = Boolean($("#taskTime").value);
    document.querySelectorAll("[data-remind][data-timed]").forEach(b => {
      b.hidden = !timed;
      if (!timed && b.getAttribute("aria-pressed") === "true") {
        b.setAttribute("aria-pressed", "false");
        document.querySelector('[data-remind][data-value="0"]').setAttribute("aria-pressed", "true");
      }
    });
    // Новой задаче со временем по умолчанию ставим напоминание «в срок».
    if (!remindTouched) {
      const value = timed ? "0" : "none";
      document.querySelectorAll("[data-remind]").forEach(b => b.setAttribute("aria-pressed", String(b.dataset.value === value)));
    }
  };
  $("#taskTime").addEventListener("input", syncRemindOptions);
  syncRemindOptions();
  document.querySelectorAll("[data-weekday]").forEach(b => {
    b.onclick = () => {
      b.setAttribute("aria-pressed", String(b.getAttribute("aria-pressed") !== "true"));
      setDirty(true);
    };
  });
  bindChoice("[data-tlist]");
  initChecklist(task);
  if (task) {
    $("#taskToggle").onclick = async () => {
      await closeSheet(true);
      toggleTask(task.id);
    };
    $("#taskDelete").onclick = () => deleteTask(task);
  }
}

function readRemind() {
  const v = chosen("[data-remind]");
  if (!v || v === "none" || chosen("[data-due]") === "none") return null;
  return Number(v);
}

function readRepeat() {
  const kind = chosen("[data-repeat]");
  if (!kind || kind === "none" || chosen("[data-due]") === "none") return null;
  if (kind !== "weekly") return { kind };
  const weekdays = [...document.querySelectorAll('[data-weekday][aria-pressed="true"]')].map(b => Number(b.dataset.weekday));
  return { kind, weekdays };
}

function readDue() {
  const preset = chosen("[data-due]");
  if (preset === "none") return { due_at: null, due_all_day: false };
  const now = new Date();
  const date =
    preset === "today"
      ? toDateInput(now)
      : preset === "tomorrow"
        ? toDateInput(new Date(now.getFullYear(), now.getMonth(), now.getDate() + 1))
        : $("#taskDate").value;
  return buildDue(date, $("#taskTime").value);
}

async function saveTask(task) {
  const title = $("#taskTitle").value.trim();
  const description = $("#taskDesc").value.trim();
  if (!title) {
    toast("Напишите, что нужно сделать.", true);
    $("#taskTitle").focus();
    return;
  }
  if (chosen("[data-due]") === "date" && !$("#taskDate").value) {
    toast("Выберите дату срока.", true);
    return;
  }
  const due = readDue();
  const repeat = readRepeat();
  const remind = readRemind();
  if (repeat?.kind === "weekly" && !repeat.weekdays.length) {
    toast("Выберите хотя бы один день недели.", true);
    return;
  }
  const listId = Number(chosen("[data-tlist]"));
  try {
    if (task) {
      const body = {};
      if (title !== task.title) body.title = title;
      if (description !== (task.description || "")) body.description = description || null;
      if (listId !== task.list_id) body.list_id = listId;
      if (due.due_at !== task.due_at || due.due_all_day !== task.due_all_day) {
        // Сравниваем моменты, а не строки: сервер может вернуть другой формат.
        const same =
          due.due_all_day === task.due_all_day &&
          ((!due.due_at && !task.due_at) || (due.due_at && task.due_at && +new Date(due.due_at) === +new Date(task.due_at)));
        if (!same) Object.assign(body, due);
      }
      if (!sameRepeat(repeat, task.repeat)) body.repeat = repeat;
      if (remind !== (task.remind_before_minutes ?? null)) body.remind_before_minutes = remind;
      if (!Object.keys(body).length) return closeSheet(true);
      body.version = task.version;
      await api(`/tasks/${task.id}`, send("PATCH", body));
      toast("Задача сохранена");
    } else {
      const created = await api("/tasks", send("POST", { title, description: description || undefined, list_id: listId, ...due, repeat, remind_before_minutes: remind }), {
        404: "Список не найден — возможно, его удалили.",
      });
      await savePendingItems(created.id);
      haptic("success");
      toast("Задача создана");
    }
    await closeSheet(true);
    await refreshListsAndTasks();
  } catch (err) {
    if (err.status === 409 || err.status === 404) await closeSheet(true);
    handleError(err, refreshListsAndTasks);
  }
}

// ---------- Чеклист ----------
// У существующей задачи пункты сохраняются сразу, у новой — копятся в форме
// и отправляются после создания задачи.
let checklist = null;
const HIDE_DONE_KEY = "todo-hide-done-items";

function readHideDone() {
  try {
    return localStorage.getItem(HIDE_DONE_KEY) === "1";
  } catch {
    return false;
  }
}

async function initChecklist(task) {
  const box = $("#checklist");
  checklist = { task, items: [], hideDone: readHideDone(), changed: false, offered: false };
  // Ввод пункта не делает форму задачи «несохранённой».
  box.addEventListener("input", e => e.stopPropagation());
  if (task) {
    box.innerHTML = '<p class="hint">Загружаем пункты…</p>';
    try {
      const items = (await api(`/tasks/${task.id}/items`)) || [];
      if (checklist?.task !== task) return;
      checklist.items = items;
      // Уже всё отмечено — не предлагаем выполнить, пока что-то не изменится.
      checklist.offered = items.length > 0 && items.every(i => i.done);
    } catch (err) {
      if (checklist?.task === task) box.innerHTML = `<p class="hint">${esc(err.message)}</p>`;
      return;
    }
  }
  renderChecklist();
}

function renderChecklist() {
  const box = $("#checklist");
  if (!box || !checklist) return;
  const { items, hideDone } = checklist;
  const done = items.filter(i => i.done).length;
  const visible = hideDone ? items.filter(i => !i.done) : items;
  box.innerHTML = `${visible
    .map(
      i => `<div class="check-item${i.done ? " done" : ""}">
        <button class="task-check" type="button" data-item-check="${i.id}" aria-pressed="${i.done}" aria-label="${
          i.done ? "Снять отметку" : "Отметить"
        }: ${esc(i.title)}">${i.done ? icon("check", 14) : ""}</button>
        <span class="check-title">${esc(i.title)}</span>
        <button class="icon-btn" type="button" data-item-delete="${i.id}" aria-label="Удалить пункт: ${esc(i.title)}">${icon("close", 16)}</button>
      </div>`,
    )
    .join("")}
    <div class="inline check-add">
      <input class="field" id="itemTitle" maxlength="200" placeholder="Добавить пункт" aria-label="Новый пункт" />
      <button class="btn quiet" type="button" id="itemAdd" aria-label="Добавить пункт">${icon("plus", 18)}</button>
    </div>
    ${
      done
        ? `<button class="btn quiet block check-hide" type="button" id="itemsHideDone" aria-pressed="${hideDone}">${
            hideDone ? `Показать отмеченные (${done})` : "Скрыть отмеченные"
          }</button>`
        : ""
    }`;

  box.querySelectorAll("[data-item-check]").forEach(b => (b.onclick = () => toggleItem(Number(b.dataset.itemCheck))));
  box.querySelectorAll("[data-item-delete]").forEach(b => (b.onclick = () => deleteItem(Number(b.dataset.itemDelete))));
  $("#itemAdd").onclick = () => addItem();
  $("#itemTitle").addEventListener("keydown", e => {
    if (e.key !== "Enter") return;
    e.preventDefault();
    addItem();
  });
  if (done) {
    $("#itemsHideDone").onclick = () => {
      checklist.hideDone = !checklist.hideDone;
      try {
        localStorage.setItem(HIDE_DONE_KEY, checklist.hideDone ? "1" : "0");
      } catch {
        /* без памяти — только до закрытия */
      }
      renderChecklist();
    };
  }
}

// На 409/404 перечитываем пункты; если пропала сама задача — закрываем лист.
async function checklistError(err) {
  toast(err.message, true);
  if (err.status !== 409 && err.status !== 404) return;
  const task = checklist?.task;
  try {
    const items = (await api(`/tasks/${task.id}/items`)) || [];
    if (checklist?.task !== task) return;
    checklist.items = items;
    renderChecklist();
  } catch (e) {
    if (e.status === 404) {
      await closeSheet(true);
      await refreshListsAndTasks();
    }
  }
}

async function addItem() {
  const input = $("#itemTitle");
  const title = input.value.trim();
  if (!title) return input.focus();
  const { task } = checklist;
  if (!task) {
    checklist.items.push({ id: -(checklist.items.length + 1), title, done: false });
    setDirty(true);
  } else {
    try {
      const item = await api(`/tasks/${task.id}/items`, send("POST", { title }), { 409: "В задаче уже 100 пунктов." });
      if (checklist?.task !== task) return;
      checklist.items.push(item);
      checklist.changed = true;
      checklist.offered = false;
    } catch (err) {
      return checklistError(err);
    }
  }
  renderChecklist();
  $("#itemTitle").focus();
}

async function toggleItem(id) {
  const item = checklist.items.find(i => i.id === id);
  if (!item) return;
  const { task } = checklist;
  if (!task) {
    item.done = !item.done;
    return renderChecklist();
  }
  try {
    const updated = await api(`/tasks/${task.id}/items/${id}`, send("PATCH", { done: !item.done, version: item.version }));
    if (checklist?.task !== task) return;
    Object.assign(item, updated);
    checklist.changed = true;
    renderChecklist();
    offerComplete();
  } catch (err) {
    checklistError(err);
  }
}

async function deleteItem(id) {
  const { task } = checklist;
  if (task) {
    try {
      await api(`/tasks/${task.id}/items/${id}`, send("DELETE"));
    } catch (err) {
      return checklistError(err);
    }
    if (checklist?.task !== task) return;
    checklist.changed = true;
  }
  checklist.items = checklist.items.filter(i => i.id !== id);
  renderChecklist();
  offerComplete();
}

// Все пункты отмечены — предлагаем выполнить задачу, но сами не закрываем.
async function offerComplete() {
  const { task, items } = checklist;
  const allDone = items.length > 0 && items.every(i => i.done);
  if (!allDone) {
    checklist.offered = false;
    return;
  }
  if (!task || task.completed || checklist.offered) return;
  checklist.offered = true;
  haptic("success");
  if (!(await ask("Все пункты отмечены. Выполнить задачу?"))) return;
  await closeSheet();
  if (!isSheetOpen()) toggleTask(task.id);
}

// Пункты новой задачи — после её создания, по порядку.
async function savePendingItems(taskId) {
  for (const i of checklist?.items || []) {
    const item = await api(`/tasks/${taskId}/items`, send("POST", { title: i.title }));
    if (i.done) await api(`/tasks/${taskId}/items/${item.id}`, send("PATCH", { done: true, version: item.version }));
  }
}

async function deleteTask(task) {
  if (!(await ask("Удалить задачу?"))) return;
  try {
    await api(`/tasks/${task.id}`, send("DELETE"));
    await closeSheet(true);
    toast("Задача удалена");
    await refreshListsAndTasks();
  } catch (err) {
    await closeSheet(true);
    handleError(err, refreshListsAndTasks);
  }
}

// ---------- Форма списка ----------
function membersBlock(list) {
  if (list.is_default) return "";
  const owner = isOwner(list);
  const rows = (list.members || [])
    .map(m => {
      const you = m.user_id === state.me.id;
      const sub = m.role === "owner" ? (you ? "вы, владелец" : "владелец") : you ? "вы" : "";
      const canRemove = owner && m.role !== "owner";
      return `<div class="row"><span class="row-main">${esc(m.full_name)}${sub ? `<div class="row-sub">${sub}</div>` : ""}</span>${
        canRemove
          ? `<button class="icon-btn" type="button" data-remove-member="${m.user_id}" aria-label="Исключить ${esc(m.full_name)}">${icon("close", 18)}</button>`
          : ""
      }</div>`;
    })
    .join("");
  // Переключатель у каждого свой: и у владельца, и у участников.
  const notify = isShared(list)
    ? `<div class="rows" style="background:var(--field);margin-top:8px"><div class="row">
        <span class="row-main">Сообщать об изменениях<div class="row-sub">когда другие добавляют или выполняют задачи</div></span>
        <button class="switch" type="button" role="switch" id="notifySwitch" aria-checked="${list.notify_changes !== false}" aria-label="Сообщать об изменениях"></button>
      </div></div>`
    : "";
  return `<span class="field-label">Участники</span>
    <div class="rows" style="background:var(--field)">${rows}</div>
    ${notify}
    ${owner ? '<div id="inviteBlock" style="margin-top:8px"></div>' : ""}`;
}

function renderInviteBlock(list) {
  const block = $("#inviteBlock");
  if (!block) return;
  if (!list.invite_code) {
    block.innerHTML = `<button class="btn quiet block" type="button" id="inviteCreate">${icon("users", 18)} Пригласить по ссылке</button>`;
    $("#inviteCreate").onclick = () => createInvite(list);
    return;
  }
  block.innerHTML = list.invite_link
    ? `<div class="inline"><button class="btn primary" type="button" id="inviteShare">${icon("share", 18)} Поделиться</button>
        <button class="btn quiet" type="button" id="inviteCopy">${icon("copy", 18)} Скопировать</button></div>
       <p class="hint">Кто откроет ссылку в Telegram, попадёт в этот список.</p>
       <button class="btn danger block" type="button" id="inviteRevoke">Сбросить ссылку</button>`
    : `<p class="hint">Приглашение создано, но ссылку не собрать: на сервере не указано имя бота (AUTH_TELEGRAM_BOT_USERNAME).</p>
       <button class="btn danger block" type="button" id="inviteRevoke">Выключить приглашение</button>`;
  if (list.invite_link) {
    $("#inviteShare").onclick = () => shareInvite(list);
    $("#inviteCopy").onclick = () => copyInvite(list);
  }
  $("#inviteRevoke").onclick = () => revokeInvite(list);
}

function openListSheet(list = null) {
  const owner = !list || isOwner(list);
  const color = list?.color || LIST_COLORS.find(c => !state.lists.some(l => l.color === c)) || "blue";
  const disabled = owner ? "" : "disabled";
  openSheet(
    list ? "Список" : "Новый список",
    `<form id="listForm" novalidate>
      <input class="field" id="listTitle" maxlength="50" placeholder="Например, «Дом»" value="${esc(list?.title)}" aria-label="Название списка" ${disabled} />
      <span class="field-label">Цвет</span>
      <div class="palette" role="group" aria-label="Цвет списка">${LIST_COLORS.map(
        c =>
          `<button class="swatch" type="button" data-color="" data-value="${c}" aria-pressed="${c === color}" aria-label="${
            COLOR_NAMES[c]
          }" ${colorStyle(c)} ${disabled}></button>`,
      ).join("")}</div>
      ${list ? membersBlock(list) : ""}
      <div class="sheet-actions">
        ${owner ? `<button class="btn primary block" type="submit">${list ? "Сохранить" : "Создать список"}</button>` : ""}
        ${
          list && owner && !list.is_default
            ? `<button class="btn danger block" type="button" id="listDelete">Удалить список${
                list.total_tasks ? ` и ${list.total_tasks} ${plural(list.total_tasks, "задачу", "задачи", "задач")}` : ""
              }</button>`
            : ""
        }
        ${list && !owner ? '<button class="btn danger block" type="button" id="listLeave">Выйти из списка</button>' : ""}
      </div>
      ${list?.is_default ? '<p class="hint">Это список по умолчанию: его нельзя удалить или сделать общим.</p>' : ""}
      ${list && !owner ? '<p class="hint">Название и цвет меняет владелец списка.</p>' : ""}
    </form>`,
    { onSubmit: owner ? () => saveList(list) : null },
  );
  if (owner) bindChoice("[data-color]");
  if (!list) return;
  if (owner && !list.is_default) {
    $("#listDelete").onclick = () => deleteList(list);
    renderInviteBlock(list);
    document.querySelectorAll("[data-remove-member]").forEach(b => {
      b.onclick = () => removeMember(list, Number(b.dataset.removeMember));
    });
  }
  if (!owner) $("#listLeave").onclick = () => leaveList(list);
  if ($("#notifySwitch")) $("#notifySwitch").onclick = () => toggleListNotify(list);
}

async function toggleListNotify(list) {
  const on = list.notify_changes === false;
  if (on && !(await requestWriteAccess())) {
    toast("Без разрешения бот не сможет присылать сообщения.", true);
    return;
  }
  const sw = $("#notifySwitch");
  sw.setAttribute("aria-checked", String(on));
  try {
    await api(`/lists/${list.id}/notifications`, send("PUT", { notify_changes: on }));
    list.notify_changes = on;
    toast(on ? "Бот напишет, когда в списке что-то изменится" : "Сообщения об изменениях выключены");
  } catch (err) {
    sw.setAttribute("aria-checked", String(!on));
    toast(err.message, true);
  }
}

async function createInvite(list) {
  try {
    const updated = await api(`/lists/${list.id}/invite`, send("POST"), { 409: "Список по умолчанию нельзя сделать общим." });
    Object.assign(list, { invite_code: updated.invite_code, invite_link: updated.invite_link });
    renderInviteBlock(list);
  } catch (err) {
    toast(err.message, true);
  }
}

async function revokeInvite(list) {
  if (!(await ask("Старая ссылка перестанет работать. Уже вступившие останутся в списке."))) return;
  try {
    await api(`/lists/${list.id}/invite`, send("DELETE"));
    Object.assign(list, { invite_code: null, invite_link: null });
    renderInviteBlock(list);
    toast("Ссылка сброшена");
  } catch (err) {
    toast(err.message, true);
  }
}

function shareInvite(list) {
  const text = `Присоединяйся к списку «${list.title}»`;
  if (inTelegram && tg.openTelegramLink) {
    tg.openTelegramLink(
      `https://t.me/share/url?url=${encodeURIComponent(list.invite_link)}&text=${encodeURIComponent(text)}`,
    );
  } else if (navigator.share) {
    navigator.share({ title: text, url: list.invite_link }).catch(() => {});
  } else {
    copyInvite(list);
  }
}

async function copyInvite(list) {
  try {
    await navigator.clipboard.writeText(list.invite_link);
    toast("Ссылка скопирована");
  } catch {
    toast(list.invite_link);
  }
}

async function removeMember(list, userId) {
  const name = memberName(list, userId) || "участника";
  if (!(await ask(`Исключить ${name} из списка «${list.title}»?`))) return;
  try {
    await api(`/lists/${list.id}/members/${userId}`, send("DELETE"));
    await closeSheet(true);
    toast("Участник исключён");
    await afterListsChanged();
  } catch (err) {
    await closeSheet(true);
    handleError(err, afterListsChanged);
  }
}

async function leaveList(list) {
  if (!(await ask(`Выйти из списка «${list.title}»? Задачи в нём останутся у других участников.`))) return;
  try {
    await api(`/lists/${list.id}/members/${state.me.id}`, send("DELETE"));
    if (state.listFilter === list.id) state.listFilter = "all";
    await closeSheet(true);
    toast("Вы вышли из списка");
    await afterListsChanged();
  } catch (err) {
    await closeSheet(true);
    handleError(err, afterListsChanged);
  }
}

async function saveList(list) {
  const title = $("#listTitle").value.trim();
  const color = chosen("[data-color]");
  if (!title) {
    toast("Назовите список.", true);
    $("#listTitle").focus();
    return;
  }
  try {
    if (list) {
      await api(`/lists/${list.id}`, send("PATCH", { title, color, version: list.version }));
      toast("Список сохранён");
    } else {
      const created = await api("/lists", send("POST", { title, color }));
      state.listFilter = created.id;
      toast("Список создан");
    }
    await closeSheet(true);
    await afterListsChanged();
  } catch (err) {
    if (err.status === 409 || err.status === 404) await closeSheet(true);
    handleError(err, afterListsChanged);
  }
}

async function deleteList(list) {
  const tasks = list.total_tasks
    ? ` и ${list.total_tasks} ${plural(list.total_tasks, "задачу", "задачи", "задач")} в нём`
    : "";
  const shared = isShared(list) ? " Участники тоже потеряют доступ." : "";
  if (!(await ask(`Удалить список «${list.title}»${tasks}?${shared}`))) return;
  try {
    await api(`/lists/${list.id}`, send("DELETE"), { 409: "Список по умолчанию удалить нельзя." });
    if (state.listFilter === list.id) state.listFilter = "all";
    await closeSheet(true);
    toast("Список удалён");
    await afterListsChanged();
  } catch (err) {
    await closeSheet(true);
    handleError(err, afterListsChanged);
  }
}

async function afterListsChanged() {
  await loadLists();
  if (state.tab === "tasks") await loadTasks();
  else if (state.tab === "profile") renderProfile();
}

// ---------- Статистика ----------
const PERIODS = { week: "Неделя", month: "Месяц", all: "Всё время" };
async function loadStats() {
  renderLoader();
  const params = new URLSearchParams();
  const userId = isAdmin() && state.statsUser ? state.statsUser : state.me.id;
  params.set("user_id", userId);
  if (state.statsPeriod !== "all") {
    const days = state.statsPeriod === "week" ? 7 : 30;
    const from = new Date();
    from.setDate(from.getDate() - days + 1);
    params.set("from", toDateInput(from));
  }
  try {
    const [stats] = await Promise.all([
      api(`/statistics?${params}`),
      isAdmin() && !state.users.length ? loadUsersData() : null,
    ]);
    renderStats(stats);
  } catch (e) {
    renderError(e, loadStats);
  }
}

function renderStats(s) {
  const created = s.tasks_created || 0,
    completed = s.tasks_completed || 0;
  const onTime = s.tasks_on_time_rate == null ? "—" : `${Math.round(s.tasks_on_time_rate)}%`;
  const lists = s.lists || [];
  $("#screen").innerHTML = `
    <div class="screen-head"><h1 class="title">Статистика</h1></div>
    ${
      isAdmin()
        ? `<select class="field" id="statsUser" aria-label="Чья статистика" style="margin-bottom:10px">
            <option value="">Моя</option>${state.users
              .filter(u => u.id !== state.me.id)
              .map(u => `<option value="${u.id}" ${String(u.id) === String(state.statsUser) ? "selected" : ""}>${esc(u.full_name)}</option>`)
              .join("")}</select>`
        : ""
    }
    <div class="segmented" role="group" aria-label="Период">${Object.entries(PERIODS)
      .map(([k, v]) => `<button type="button" data-period="${k}" aria-pressed="${state.statsPeriod === k}">${v}</button>`)
      .join("")}</div>
    ${
      created
        ? `<div class="stat-grid">
            <div class="card"><div class="stat-label">Выполнено</div><div class="stat-value">${completed}<small> из ${created}</small></div></div>
            <div class="card"><div class="stat-label">В срок</div><div class="stat-value">${onTime}</div></div>
            <div class="card wide"><div class="stat-label">Обычно задача занимает</div><div class="stat-value" style="font-size:20px">${fmtDuration(
              s.tasks_average_completion_seconds,
            )}</div></div>
          </div>
          ${
            lists.length
              ? `<div class="section-label">По спискам</div><div class="card">${lists
                  .map(
                    l => `<div class="bar-row" ${colorStyle(l.color)}>
                      <div class="bar-head"><span class="dot"></span><span class="row-main">${esc(l.title)}</span><span class="row-sub">${
                        l.tasks_completed
                      } из ${l.tasks_created}</span></div>
                      <div class="bar-track"><div class="bar-fill" style="width:${
                        l.tasks_created ? Math.round((l.tasks_completed / l.tasks_created) * 100) : 0
                      }%"></div></div></div>`,
                  )
                  .join("")}</div>`
              : ""
          }`
        : `<div class="empty"><h2>Пока нечего считать</h2><p>За этот период задач не было.</p></div>`
    }`;
  document.querySelectorAll("[data-period]").forEach(b => {
    b.onclick = () => {
      state.statsPeriod = b.dataset.period;
      loadStats();
    };
  });
  if (isAdmin())
    $("#statsUser").onchange = e => {
      state.statsUser = e.target.value;
      loadStats();
    };
}

// ---------- Профиль ----------
function avatarStyle(id, extra = "") {
  const c = LIST_COLORS[id % LIST_COLORS.length];
  return `style="--avatar-bg: color-mix(in srgb, var(--c-${c}) 18%, var(--surface)); --avatar-fg: var(--c-${c}); ${extra}"`;
}

async function renderProfile() {
  renderLoader();
  try {
    await loadLists();
  } catch (e) {
    return renderError(e, renderProfile);
  }
  const me = state.me;
  const dark = document.body.classList.contains("dark");
  $("#screen").innerHTML = `
    <div class="profile">
      <div class="avatar" ${avatarStyle(me.id)} aria-hidden="true">${esc(initials(me.full_name))}</div>
      <div class="row-main"><div class="profile-name">${esc(me.full_name)}</div><div class="row-sub">${esc(
        me.phone_number || "Телефон не указан",
      )}</div></div>
      <button class="icon-btn" type="button" id="editProfile" aria-label="Изменить профиль">${icon("edit")}</button>
    </div>

    <div class="section-label">Уведомления</div>
    <div class="rows">
      <div class="row"><span class="row-main">Напоминания о сроках<div class="row-sub">когда указано у задачи</div></span>
        <button class="switch" type="button" role="switch" id="remindSwitch" aria-checked="${Boolean(me.remind_enabled)}" aria-label="Напоминания о сроках"></button></div>
      <div class="row"><span class="row-main">Утренняя сводка<div class="row-sub">что на сегодня и что просрочено</div></span>
        <button class="switch" type="button" role="switch" id="digestSwitch" aria-checked="${Boolean(me.digest_enabled)}" aria-label="Утренняя сводка"></button></div>
      <label class="row" for="digestTime"><span class="row-main">Время сводки<div class="row-sub">и напоминаний по задачам на весь день</div></span>
        <input class="field" type="time" id="digestTime" value="${esc(me.digest_time || "09:00")}" style="width:auto;padding:6px 10px" /></label>
      <div class="row"><span class="row-main">Часовой пояс</span><span class="row-sub">${esc(tzName(me.timezone))}</span></div>
    </div>

    <div class="section-label">Мои списки</div>
    <div class="rows">${state.lists
      .map(
        l => `<button class="row" type="button" data-edit-list="${l.id}" ${colorStyle(l.color)}>
          <span class="dot" style="width:10px;height:10px"></span>
          <span class="row-main">${esc(l.title)}</span>
          <span class="row-sub">${isShared(l) ? `${icon("users", 14)} ${l.members.length} · ` : ""}${l.open_tasks} ${plural(
            l.open_tasks,
            "задача",
            "задачи",
            "задач",
          )}</span>
          <span class="chev">${icon("chev", 18)}</span></button>`,
      )
      .join("")}
      <button class="row" type="button" id="newListRow"><span class="row-main">${icon("plus", 18)} Новый список</span></button>
    </div>

    ${
      inTelegram
        ? ""
        : `<div class="section-label">Оформление</div>
          <div class="rows"><div class="row"><span class="row-main">Тёмная тема</span>
          <button class="switch" type="button" role="switch" id="themeSwitch" aria-checked="${dark}" aria-label="Тёмная тема"></button></div></div>`
    }

    ${
      isAdmin()
        ? `<div class="section-label">Администрирование</div>
          <div class="rows"><button class="row" type="button" id="usersRow">${icon("users", 18)}<span class="row-main">Все пользователи</span><span class="chev">${icon(
            "chev",
            18,
          )}</span></button></div>`
        : ""
    }`;

  $("#editProfile").onclick = () => openUserSheet(me, { self: true });
  $("#remindSwitch").onclick = () => toggleNotification("remind_enabled", "#remindSwitch");
  $("#digestSwitch").onclick = () => toggleNotification("digest_enabled", "#digestSwitch");
  $("#digestTime").onchange = e => e.target.value && saveSettings({ digest_time: e.target.value }, "Время сводки сохранено");
  document.querySelectorAll("[data-edit-list]").forEach(b => {
    b.onclick = () => openListSheet(listById(Number(b.dataset.editList)));
  });
  $("#newListRow").onclick = () => openListSheet();
  if (!inTelegram)
    $("#themeSwitch").onclick = () => {
      const on = !document.body.classList.contains("dark");
      document.body.classList.toggle("dark", on);
      try {
        localStorage.setItem("todo-theme", on ? "dark" : "light");
      } catch {}
      $("#themeSwitch").setAttribute("aria-checked", String(on));
    };
  if (isAdmin()) $("#usersRow").onclick = () => switchTab("users");
}

// ---------- Уведомления ----------
const TZ_CITIES = {
  "Europe/Kaliningrad": "Калининград",
  "Europe/Moscow": "Москва",
  "Europe/Samara": "Самара",
  "Europe/Volgograd": "Волгоград",
  "Asia/Yekaterinburg": "Екатеринбург",
  "Asia/Omsk": "Омск",
  "Asia/Novosibirsk": "Новосибирск",
  "Asia/Krasnoyarsk": "Красноярск",
  "Asia/Irkutsk": "Иркутск",
  "Asia/Yakutsk": "Якутск",
  "Asia/Vladivostok": "Владивосток",
  "Asia/Magadan": "Магадан",
  "Asia/Kamchatka": "Камчатка",
};

function tzName(tz) {
  if (!tz || tz === "UTC") return "UTC";
  const city = TZ_CITIES[tz] || tz.split("/").pop().replace(/_/g, " ");
  try {
    const offset = new Intl.DateTimeFormat("ru-RU", { timeZone: tz, timeZoneName: "shortOffset" })
      .formatToParts(new Date())
      .find(p => p.type === "timeZoneName")?.value;
    return offset ? `${city}, ${offset.replace("GMT", "UTC")}` : city;
  } catch {
    return city;
  }
}

// Бот может писать человеку, только если тот разрешил. Спрашиваем при включении.
function requestWriteAccess() {
  if (!inTelegram || !tg.requestWriteAccess || !tg.isVersionAtLeast?.("6.9")) return Promise.resolve(true);
  return new Promise(resolve => tg.requestWriteAccess(granted => resolve(Boolean(granted))));
}

async function toggleNotification(field, switchSel) {
  const on = !state.me[field];
  if (on && !(await requestWriteAccess())) {
    toast("Без разрешения бот не сможет присылать сообщения.", true);
    return;
  }
  $(switchSel).setAttribute("aria-checked", String(on));
  const ok = await saveSettings({ [field]: on }, on ? "Включено" : "Выключено");
  if (!ok) $(switchSel).setAttribute("aria-checked", String(!on));
}

async function saveSettings(body, message) {
  try {
    const updated = await api(`/users/${state.me.id}`, send("PATCH", { ...body, version: state.me.version }));
    state.me = { ...state.me, ...updated };
    toast(message);
    return true;
  } catch (err) {
    toast(err.message, true);
    if (err.status === 409) await refreshMe();
    return false;
  }
}

// ---------- Пользователь (свой профиль и админка) ----------
function openUserSheet(user, { self = false } = {}) {
  openSheet(
    self ? "Профиль" : "Пользователь",
    `<form id="userForm" novalidate>
      <label class="field-label" for="userName" style="margin-top:4px">Имя</label>
      <input class="field" id="userName" maxlength="100" value="${esc(user.full_name)}" />
      <label class="field-label" for="userPhone">Телефон</label>
      <input class="field" id="userPhone" type="tel" inputmode="tel" maxlength="15" placeholder="+79991234567" value="${esc(
        user.phone_number,
      )}" />
      <p class="hint">Необязательно. Плюс и от 9 до 14 цифр.</p>
      <div class="sheet-actions">
        <button class="btn primary block" type="submit">Сохранить</button>
        ${!self && user.id !== state.me.id ? '<button class="btn danger block" type="button" id="userDelete">Удалить пользователя</button>' : ""}
      </div>
    </form>`,
    { onSubmit: () => saveUser(user, self) },
  );
  if (!self && user.id !== state.me.id) $("#userDelete").onclick = () => deleteUser(user);
}

async function saveUser(user, self) {
  const full_name = $("#userName").value.trim();
  const phone = $("#userPhone").value.trim();
  if (!full_name) {
    toast("Укажите имя.", true);
    return;
  }
  if (phone && !/^\+[0-9]{9,14}$/.test(phone)) {
    toast("Телефон: плюс и от 9 до 14 цифр, например +79991234567.", true);
    return;
  }
  const body = {};
  if (full_name !== user.full_name) body.full_name = full_name;
  // Пустое поле — удалить телефон: отправляем null, а не "".
  if (phone !== (user.phone_number || "")) body.phone_number = phone || null;
  if (!Object.keys(body).length) return closeSheet(true);
  body.version = user.version;
  try {
    const updated = await api(`/users/${user.id}`, send("PATCH", body));
    if (updated.id === state.me.id) state.me = { ...state.me, ...updated };
    await closeSheet(true);
    toast("Сохранено");
    if (self) renderProfile();
    else loadUsers();
  } catch (err) {
    await closeSheet(true);
    handleError(err, self ? refreshMe : loadUsers);
  }
}

async function refreshMe() {
  state.me = await api("/me");
  renderProfile();
}

async function loadUsersData() {
  const users = await api("/users?limit=500&offset=0");
  state.users = Array.isArray(users) ? users : [];
}

async function loadUsers() {
  renderLoader();
  try {
    await loadUsersData();
  } catch (e) {
    return renderError(e, loadUsers);
  }
  $("#screen").innerHTML = `
    <div class="screen-head">
      ${inTelegram ? "" : `<button class="icon-btn" type="button" id="usersBack" aria-label="Назад">${icon("back")}</button>`}
      <h1 class="title">Пользователи</h1>
    </div>
    <div class="rows">${state.users
      .map(
        u => `<button class="row" type="button" data-user="${u.id}">
          <div class="avatar" ${avatarStyle(u.id, "width:36px;height:36px;font-size:13px")} aria-hidden="true">${esc(initials(u.full_name))}</div>
          <span class="row-main">${esc(u.full_name)}<div class="row-sub">${u.telegram_id ? "Telegram" : "без Telegram"}${
            u.phone_number ? `, ${esc(u.phone_number)}` : ""
          }</div></span>
          <span class="chev">${icon("chev", 18)}</span></button>`,
      )
      .join("")}</div>`;
  if (!inTelegram) $("#usersBack").onclick = () => switchTab("profile");
  document.querySelectorAll("[data-user]").forEach(b => {
    b.onclick = () => openUserSheet(state.users.find(u => u.id === Number(b.dataset.user)));
  });
}

async function deleteUser(user) {
  if (!(await ask(`Удалить пользователя «${user.full_name}»?`))) return;
  try {
    await api(`/users/${user.id}`, send("DELETE"), {
      409: "У пользователя есть задачи — сначала удалите их.",
    });
    await closeSheet(true);
    toast("Пользователь удалён");
    loadUsers();
  } catch (err) {
    await closeSheet(true);
    handleError(err, loadUsers);
  }
}

// ---------- Тема и Telegram ----------
function applyTheme() {
  if (inTelegram) {
    document.body.classList.toggle("dark", tg.colorScheme === "dark");
  } else {
    let saved = null;
    try {
      saved = localStorage.getItem("todo-theme");
    } catch {}
    const prefersDark = window.matchMedia?.("(prefers-color-scheme: dark)").matches;
    document.body.classList.toggle("dark", saved === "dark" || (!saved && prefersDark));
  }
  if (inTelegram) syncTelegramColors();
}

// Шапка, фон и системная кнопка Telegram в цветах приложения.
function syncTelegramColors() {
  const css = getComputedStyle(document.body);
  const bg = css.getPropertyValue("--bg").trim();
  const ink = css.getPropertyValue("--ink").trim();
  const onInk = css.getPropertyValue("--on-ink").trim();
  if (tg.isVersionAtLeast?.("6.1")) {
    tg.setHeaderColor(bg);
    tg.setBackgroundColor(bg);
  }
  if (tg.isVersionAtLeast?.("7.10")) tg.setBottomBarColor?.(bg);
  tg.MainButton.setParams({ text: "Добавить задачу", color: ink, text_color: onInk });
}

async function start() {
  $("#addTaskBtn").innerHTML = `${icon("plus", 18)} Добавить задачу`;
  $("#addTaskBtn").onclick = () => openTaskSheet();
  if (tg) {
    tg.ready();
    tg.expand();
    if (inTelegram) {
      tg.onEvent("themeChanged", applyTheme);
      tg.BackButton.onClick(() => (isSheetOpen() ? closeSheet() : switchTab("profile")));
      tg.MainButton.onClick(() => openTaskSheet());
    }
  }
  applyTheme();
  renderTabbar();
  renderLoader();
  try {
    state.me = await api("/me");
  } catch (e) {
    $("#tabbar").hidden = true;
    renderError(
      e.status === 401
        ? new Error("Откройте приложение кнопкой в Telegram-боте. Вне Telegram оно доступно только на компьютере, где запущен сервер.")
        : e,
      e.status === 401 ? null : start,
    );
    return;
  }
  $("#tabbar").hidden = false;
  await syncTimezone();
  await joinFromLink();
  await openListFromLink();
  await switchTab("tasks");
  await openTaskFromLink();
}

// Кнопка «Открыть список» в сообщении об изменениях ведёт на t.me/<бот>?startapp=list_<id>.
async function openListFromLink() {
  const param = (inTelegram ? tg.initDataUnsafe?.start_param : null) || "";
  const id = param.startsWith("list_") ? Number(param.slice(5)) : Number(new URLSearchParams(location.search).get("list"));
  if (!id) return;
  try {
    await loadLists();
  } catch {
    return;
  }
  if (listById(id)) state.listFilter = id;
  else toast("Список не найден — возможно, вас из него исключили.", true);
}

// Кнопка «Открыть задачу» в напоминании ведёт на t.me/<бот>?startapp=task_<id>.
async function openTaskFromLink() {
  const param = (inTelegram ? tg.initDataUnsafe?.start_param : null) || "";
  const id = param.startsWith("task_") ? Number(param.slice(5)) : Number(new URLSearchParams(location.search).get("task"));
  if (!id) return;
  try {
    openTaskSheet(state.tasks.find(t => t.id === id) || (await api(`/tasks/${id}`)));
  } catch (err) {
    toast(err.status === 404 ? "Задача не найдена — возможно, её удалили." : err.message, true);
  }
}

// Часовой пояс телефона нужен серверу для повторов и времени уведомлений.
// Ошибка не мешает работе: попробуем при следующем открытии.
async function syncTimezone() {
  let tz = "";
  try {
    tz = Intl.DateTimeFormat().resolvedOptions().timeZone || "";
  } catch {}
  if (!tz || tz === state.me.timezone) return;
  try {
    state.me = { ...state.me, ...(await api(`/users/${state.me.id}`, send("PATCH", { timezone: tz, version: state.me.version }))) };
  } catch {}
}

// Ссылка t.me/<бот>?startapp=join_<код> открывает мини-апп с start_param.
// Вне Telegram для проверки можно открыть /?join=<код>.
async function joinFromLink() {
  const param = (inTelegram ? tg.initDataUnsafe?.start_param : null) || "";
  const code = param.startsWith("join_") ? param.slice(5) : new URLSearchParams(location.search).get("join");
  if (!code) return;
  try {
    const list = await api("/lists/join", send("POST", { code }), {
      404: "Ссылка-приглашение не работает: её сбросили или она неверная.",
    });
    state.listFilter = list.id;
    toast(list.owner_user_id === state.me.id ? `Это ваш список «${list.title}»` : `Вы в списке «${list.title}»`);
  } catch (err) {
    toast(err.message, true);
  }
  if (!inTelegram) history.replaceState(null, "", location.pathname);
}
start();
