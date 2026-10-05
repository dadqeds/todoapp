// Относительный путь: UI и API отдаются одним сервером, CORS не нужен.
const API_BASE = "/api/v1";
// Справочник пользователей для имён, фильтров и выпадающих списков.
// Хранится отдельно от постраничного списка на странице «Пользователи».
const DIRECTORY_LIMIT = 500;
const state = {
  page: "tasks",
  tasks: [],
  taskHasMore: false,
  taskPage: 0,
  taskLimit: 20,
  taskUser: "all",
  users: [],
  userHasMore: false,
  userPage: 0,
  userLimit: 20,
  directory: [],
  directoryLoaded: false,
  stats: null,
  statsFilter: { user: "", from: "", to: "" },
};
const $ = s => document.querySelector(s);
const esc = v =>
  String(v ?? "").replace(
    /[&<>'"]/g,
    c =>
      ({
        "&": "&amp;",
        "<": "&lt;",
        ">": "&gt;",
        "'": "&#039;",
        '"': "&quot;",
      })[c],
  );
const initials = n =>
  String(n || "?")
    .trim()
    .split(/\s+/)
    .slice(0, 2)
    .map(x => x[0])
    .join("")
    .toUpperCase();
const fmtDate = v => {
  if (!v) return "—";
  const d = new Date(v);
  if (Number.isNaN(d.getTime())) return esc(v);
  return d
    .toLocaleDateString("ru-RU", {
      day: "2-digit",
      month: "short",
      year: "numeric",
    })
    .replace(" г.", "");
};
const icon = (name, size = 16) => {
  const paths = {
    calendar:
      '<rect x="3" y="4" width="18" height="17" rx="2"/><path d="M16 2v4M8 2v4M3 9h18"/>',
    user: '<circle cx="12" cy="7" r="4"/><path d="M4 21c1-4.2 3.7-6 8-6s7 1.8 8 6"/>',
    clock: '<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>',
    trash:
      '<path d="M4 7h16M10 11v6M14 11v6"/><path d="M9 4h6l1 3H8l1-3zM6 7l1 14h10l1-14"/>',
    edit: '<path d="M4 20h4L19 9l-4-4L4 16v4z"/><path d="M13.5 6.5l4 4"/>',
    plus: '<path d="M12 5v14M5 12h14"/>',
    arrow: '<path d="M5 12h14M13 6l6 6-6 6"/>',
    check: '<path d="M5 12l4 4L19 6"/>',
  };
  return `<svg width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${paths[name] || ""}</svg>`;
};
function toast(message, error = false) {
  const t = document.createElement("div");
  t.className = "toast" + (error ? " error" : "");
  t.setAttribute("role", error ? "alert" : "status");
  t.textContent = message;
  $("#toasts").appendChild(t);
  setTimeout(() => t.remove(), 3400);
}

// Понятные сообщения по коду ответа. Сервер отдаёт технический текст
// на английском, показывать его пользователю не стоит.
const DEFAULT_MESSAGES = {
  400: "Проверьте введённые данные.",
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
        ...(options.headers || {}),
      },
    });
  } catch {
    throw new ApiError(
      0,
      "Не удалось подключиться к серверу. Проверьте, что backend запущен.",
    );
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
      (r.status >= 500
        ? "Ошибка сервера. Попробуйте позже."
        : `Ошибка запроса (${r.status}).`);
    throw new ApiError(r.status, message);
  }
  return data;
}
// Запрашиваем на одну запись больше, чтобы знать, есть ли следующая страница.
async function fetchPage(path, limit, offset) {
  const sep = path.includes("?") ? "&" : "?";
  const items = await api(
    `${path}${sep}limit=${limit + 1}&offset=${offset}`,
  );
  const list = Array.isArray(items) ? items : [];
  return { items: list.slice(0, limit), hasMore: list.length > limit };
}
async function loadDirectory(force = false) {
  if (state.directoryLoaded && !force) return state.directory;
  const users = await api(`/users?limit=${DIRECTORY_LIMIT}&offset=0`);
  state.directory = Array.isArray(users) ? users : [];
  state.directoryLoaded = true;
  return state.directory;
}
function invalidateDirectory() {
  state.directoryLoaded = false;
}
const userName = id =>
  state.directory.find(u => String(u.id) === String(id))?.full_name ||
  `Пользователь #${id}`;

function renderShell(title, subtitle, action) {
  $("#main").innerHTML =
    `<div class="topbar"><div><h1 class="page-title">${title}</h1>${subtitle ? `<div class="subtitle">${subtitle}</div>` : ""}</div>${action || ""}</div><div id="page-content"></div>`;
}
function renderLoader() {
  $("#page-content").innerHTML =
    '<div class="center"><div class="loader" aria-label="Загрузка"></div></div>';
}
function renderError(e) {
  $("#page-content").innerHTML =
    `<div class="error-box" role="alert">${esc(e.message)}</div>`;
  toast(e.message, true);
}
function setActive() {
  document.querySelectorAll(".nav button").forEach(b => {
    const active = b.dataset.page === state.page;
    b.classList.toggle("active", active);
    if (active) b.setAttribute("aria-current", "page");
    else b.removeAttribute("aria-current");
  });
}
function pager(page, hasMore, prevId, nextId) {
  return `<div class="pagination"><div>Страница ${page + 1}</div><div class="pager">
    <button class="page-btn" id="${prevId}" aria-label="Предыдущая страница" ${page === 0 ? "disabled" : ""}>‹</button>
    <button class="page-btn active" aria-current="page">${page + 1}</button>
    <button class="page-btn" id="${nextId}" aria-label="Следующая страница" ${hasMore ? "" : "disabled"}>›</button>
  </div></div>`;
}
function limitSelect(id, value) {
  return `<select class="control" id="${id}">${[10, 20, 50]
    .map(n => `<option value="${n}" ${value === n ? "selected" : ""}>${n}</option>`)
    .join("")}</select>`;
}

// ---------- Задачи ----------
function taskStatus(t) {
  return t.completed
    ? `<span class="chip status-done">${icon("check", 14)} Выполнено</span>`
    : `<span class="chip status-progress">${icon("clock", 14)} Открыта</span>`;
}
async function loadTasks() {
  renderShell(
    "Задачи",
    "Список задач с пагинацией",
    `<button class="btn primary" id="newTaskBtn">${icon("plus", 17)}&nbsp; Новая задача</button>`,
  );
  $("#newTaskBtn").onclick = () => openTaskModal();
  renderLoader();
  try {
    const limit = state.taskLimit;
    const userFilter =
      state.taskUser === "all" ? "" : `?user_id=${encodeURIComponent(state.taskUser)}`;
    const [page] = await Promise.all([
      fetchPage(`/tasks${userFilter}`, limit, state.taskPage * limit),
      loadDirectory(),
    ]);
    state.tasks = page.items;
    state.taskHasMore = page.hasMore;
    renderTasks();
  } catch (e) {
    renderError(e);
  }
}
function renderTasks() {
  const userOptions = [
    '<option value="all">Все пользователи</option>',
    ...state.directory.map(
      u =>
        `<option value="${u.id}" ${String(u.id) === String(state.taskUser) ? "selected" : ""}>${esc(u.full_name)} · ID ${u.id}</option>`,
    ),
  ].join("");
  const empty =
    state.taskUser === "all"
      ? `<h3>Задач пока нет</h3><p>Нажмите «Новая задача», чтобы создать первую</p>`
      : `<h3>У этого пользователя нет задач</h3><p>Выберите другого пользователя или сбросьте фильтр</p>`;
  $("#page-content").innerHTML = `
    <div class="filters">
      <label class="field-label" for="taskUser">Пользователь</label><select class="control" id="taskUser">${userOptions}</select>
      <label class="field-label" for="taskLimit">На странице</label>${limitSelect("taskLimit", state.taskLimit)}
      <button class="btn reset" id="resetTasks">Сбросить</button>
    </div>
    <div class="list">${
      state.tasks.length
        ? state.tasks.map(taskCard).join("")
        : `<div class="empty"><div><div class="empty-icon">${icon("check", 28)}</div>${empty}</div></div>`
    }</div>
    ${pager(state.taskPage, state.taskHasMore, "prevTask", "nextTask")}`;

  $("#taskUser").onchange = e => {
    state.taskUser = e.target.value;
    state.taskPage = 0;
    loadTasks();
  };
  $("#taskLimit").onchange = e => {
    state.taskLimit = Number(e.target.value);
    state.taskPage = 0;
    loadTasks();
  };
  $("#resetTasks").onclick = () => {
    state.taskUser = "all";
    state.taskPage = 0;
    state.taskLimit = 20;
    loadTasks();
  };
  $("#prevTask").onclick = () => {
    state.taskPage--;
    loadTasks();
  };
  $("#nextTask").onclick = () => {
    state.taskPage++;
    loadTasks();
  };
  document.querySelectorAll("[data-task-id]").forEach(card => {
    const open = () => openTaskModal(Number(card.dataset.taskId));
    card.addEventListener("click", e => {
      if (!e.target.closest("button")) open();
    });
    card.addEventListener("keydown", e => {
      if (e.target === card && (e.key === "Enter" || e.key === " ")) {
        e.preventDefault();
        open();
      }
    });
  });
  document.querySelectorAll("[data-complete-id]").forEach(
    b => (b.onclick = () => toggleTask(Number(b.dataset.completeId))),
  );
  document.querySelectorAll("[data-edit-id]").forEach(
    b => (b.onclick = () => openTaskModal(Number(b.dataset.editId))),
  );
  document.querySelectorAll("[data-delete-id]").forEach(
    b => (b.onclick = () => deleteTask(Number(b.dataset.deleteId))),
  );
}
function taskCard(t) {
  const dates = [
    `<span class="chip" title="Создана">${icon("calendar", 14)} Создана ${fmtDate(t.created_at)}</span>`,
    t.completed_at
      ? `<span class="chip" title="Выполнена">${icon("check", 14)} Выполнена ${fmtDate(t.completed_at)}</span>`
      : "",
  ].join("");
  return `<article class="task-card" data-task-id="${t.id}" tabindex="0" role="button" aria-label="Открыть задачу «${esc(t.title)}»">
    <button class="check ${t.completed ? "done" : ""}" data-complete-id="${t.id}" aria-pressed="${t.completed}" title="${t.completed ? "Вернуть в работу" : "Отметить выполненной"}" aria-label="${t.completed ? "Вернуть в работу" : "Отметить выполненной"}">${t.completed ? icon("check", 16) : ""}</button>
    <div class="task-main">
      <div class="task-title ${t.completed ? "done" : ""}">${esc(t.title)}</div>
      ${t.description ? `<div class="task-description">${esc(t.description)}</div>` : ""}
      <div class="meta"><span class="chip">${icon("user", 14)} ${esc(userName(t.author_user_id))} · ID: ${t.author_user_id}</span>${dates}${taskStatus(t)}<span class="chip" title="Версия">v${esc(t.version ?? 1)}</span></div>
    </div>
    <div class="task-actions"><button class="btn icon small" data-edit-id="${t.id}" title="Изменить" aria-label="Изменить">${icon("edit", 16)}</button><button class="btn icon small danger" data-delete-id="${t.id}" title="Удалить" aria-label="Удалить">${icon("trash", 16)}</button></div>
  </article>`;
}
// На 409 (задачу изменили в другом окне) перезагружаем список,
// чтобы пользователь увидел актуальные данные.
async function handleMutationError(err, reload) {
  toast(err.message, true);
  if (err.status === 409 || err.status === 404) await reload();
}
async function toggleTask(id) {
  const t = state.tasks.find(x => x.id === id);
  if (!t) return;
  try {
    await api(`/tasks/${id}`, {
      method: "PATCH",
      body: JSON.stringify({ completed: !t.completed, version: t.version }),
    });
    toast(t.completed ? "Задача возвращена в работу" : "Задача выполнена");
    loadTasks();
  } catch (err) {
    handleMutationError(err, loadTasks);
  }
}
async function openTaskModal(id = null) {
  let t = null;
  let users;
  try {
    [t, users] = await Promise.all([
      id ? api(`/tasks/${id}`) : null,
      loadDirectory(),
    ]);
  } catch (e) {
    handleMutationError(e, loadTasks);
    return;
  }
  if (!id && !users.length) {
    toast("Сначала создайте пользователя — задаче нужен автор.", true);
    return;
  }
  $("#modal").innerHTML = `
    <div class="modal-head"><h2 class="modal-title" id="modalTitle">${id ? "Редактировать задачу" : "Новая задача"}</h2><button class="btn icon" id="closeModal" aria-label="Закрыть">×</button></div>
    <form class="form" id="taskForm">
      ${id ? "" : `<label>Автор<select class="control" id="taskAuthor" required>${users.map(u => `<option value="${u.id}" ${String(u.id) === String(state.taskUser) ? "selected" : ""}>${esc(u.full_name)} · ID ${u.id}</option>`).join("")}</select></label>`}
      <label>Название<input class="control" id="taskTitle" maxlength="100" required value="${esc(t?.title || "")}"></label>
      <label>Описание<span class="hint"> — можно оставить пустым</span><textarea class="control" id="taskDesc" maxlength="1000">${esc(t?.description || "")}</textarea></label>
      ${id ? `<label>Статус<select class="control" id="taskCompleted"><option value="false" ${!t.completed ? "selected" : ""}>Открыта</option><option value="true" ${t.completed ? "selected" : ""}>Выполнена</option></select></label>` : ""}
      <div class="form-actions"><button type="button" class="btn" id="cancelModal">Отмена</button><button class="btn primary">${id ? "Сохранить" : "Создать задачу"}</button></div>
    </form>`;
  showModal();
  $("#taskForm").onsubmit = async e => {
    e.preventDefault();
    const title = $("#taskTitle").value.trim();
    const description = $("#taskDesc").value.trim();
    if (!title) {
      toast("Название не может быть пустым.", true);
      return;
    }
    try {
      if (id) {
        const body = {};
        if (title !== t.title) body.title = title;
        if (description !== (t.description || ""))
          body.description = description || null;
        const completed = $("#taskCompleted").value === "true";
        if (completed !== Boolean(t.completed)) body.completed = completed;
        if (!Object.keys(body).length) {
          closeModal();
          return;
        }
        body.version = t.version;
        await api(`/tasks/${id}`, {
          method: "PATCH",
          body: JSON.stringify(body),
        });
        toast("Задача обновлена");
      } else {
        await api(
          "/tasks",
          {
            method: "POST",
            body: JSON.stringify({
              author_user_id: Number($("#taskAuthor").value),
              title,
              description: description || undefined,
            }),
          },
          { 404: "Автор не найден — возможно, его удалили." },
        );
        toast("Задача создана");
      }
      closeModal();
      loadTasks();
    } catch (err) {
      if (err.status === 409 || err.status === 404) closeModal();
      handleMutationError(err, loadTasks);
    }
  };
}
async function deleteTask(id) {
  if (!confirm("Удалить эту задачу?")) return;
  try {
    await api(`/tasks/${id}`, { method: "DELETE" });
    toast("Задача удалена");
    // Удалили последнюю задачу на странице — шагаем назад.
    if (state.tasks.length === 1 && state.taskPage > 0) state.taskPage--;
    loadTasks();
  } catch (e) {
    handleMutationError(e, loadTasks);
  }
}

// ---------- Пользователи ----------
async function loadUsers() {
  renderShell(
    "Пользователи",
    "Управление пользователями",
    `<button class="btn primary" id="newUserBtn">${icon("plus", 17)}&nbsp; Новый пользователь</button>`,
  );
  $("#newUserBtn").onclick = () => openUserModal();
  renderLoader();
  try {
    const limit = state.userLimit;
    const page = await fetchPage("/users", limit, state.userPage * limit);
    state.users = page.items;
    state.userHasMore = page.hasMore;
    renderUsers();
  } catch (e) {
    renderError(e);
  }
}
function renderUsers() {
  if (!state.users.length && state.userPage === 0) {
    $("#page-content").innerHTML =
      `<div class="empty"><div><div class="empty-icon">${icon("user", 28)}</div><h3>Пользователей пока нет</h3><p>Создайте пользователя, чтобы назначать задачи</p></div></div>`;
    return;
  }
  $("#page-content").innerHTML = `
    <div class="filters">
      <label class="field-label" for="userLimit">На странице</label>${limitSelect("userLimit", state.userLimit)}
      <button class="btn reset" id="reloadUsers">Обновить</button>
    </div>
    <div class="grid">${state.users
      .map(
        u => `<article class="user-card">
          <div class="user-head"><div class="avatar" aria-hidden="true">${esc(initials(u.full_name))}</div><div><div class="user-name">${esc(u.full_name)}</div><div class="user-id">ID: ${esc(u.id)}</div></div></div>
          <div class="user-line"><span>Телефон</span><span>${esc(u.phone_number || "Не указан")}</span></div>
          <div class="user-line"><span>Версия</span><span>v${esc(u.version ?? 1)}</span></div>
          <div class="card-actions"><button class="btn small" data-user-edit="${u.id}">${icon("edit", 15)} Изменить</button><button class="btn small danger" data-user-delete="${u.id}" aria-label="Удалить пользователя ${esc(u.full_name)}" title="Удалить">${icon("trash", 15)}</button></div>
        </article>`,
      )
      .join("")}</div>
    ${pager(state.userPage, state.userHasMore, "prevUser", "nextUser")}`;
  $("#userLimit").onchange = e => {
    state.userLimit = Number(e.target.value);
    state.userPage = 0;
    loadUsers();
  };
  $("#reloadUsers").onclick = loadUsers;
  $("#prevUser").onclick = () => {
    state.userPage--;
    loadUsers();
  };
  $("#nextUser").onclick = () => {
    state.userPage++;
    loadUsers();
  };
  document
    .querySelectorAll("[data-user-edit]")
    .forEach(b => (b.onclick = () => openUserModal(Number(b.dataset.userEdit))));
  document
    .querySelectorAll("[data-user-delete]")
    .forEach(b => (b.onclick = () => deleteUser(Number(b.dataset.userDelete))));
}
async function openUserModal(id = null) {
  let u = null;
  if (id) {
    try {
      u = await api(`/users/${id}`);
    } catch (e) {
      handleMutationError(e, loadUsers);
      return;
    }
  }
  $("#modal").innerHTML = `
    <div class="modal-head"><h2 class="modal-title" id="modalTitle">${id ? "Редактировать пользователя" : "Новый пользователь"}</h2><button class="btn icon" id="closeModal" aria-label="Закрыть">×</button></div>
    <form class="form" id="userForm">
      <label>Имя и фамилия<input class="control" id="fullName" minlength="3" maxlength="100" required value="${esc(u?.full_name || "")}"></label>
      <label>Телефон<span class="hint"> — формат +79991234567, необязательное поле</span><input class="control" id="phone" type="tel" inputmode="tel" pattern="\\+[0-9]{9,14}" title="Плюс и от 9 до 14 цифр, например +79991234567" maxlength="15" value="${esc(u?.phone_number || "")}"></label>
      <div class="form-actions"><button type="button" class="btn" id="cancelModal">Отмена</button><button class="btn primary">${id ? "Сохранить" : "Создать пользователя"}</button></div>
    </form>`;
  showModal();
  $("#userForm").onsubmit = async e => {
    e.preventDefault();
    const full_name = $("#fullName").value.trim();
    const phone = $("#phone").value.trim();
    try {
      if (id) {
        const body = {};
        if (full_name !== u.full_name) body.full_name = full_name;
        // Пустое поле означает «удалить телефон» — отправляем null, а не "".
        if (phone !== (u.phone_number || "")) body.phone_number = phone || null;
        if (!Object.keys(body).length) {
          closeModal();
          return;
        }
        body.version = u.version;
        await api(`/users/${id}`, {
          method: "PATCH",
          body: JSON.stringify(body),
        });
        toast("Пользователь обновлён");
      } else {
        await api("/users", {
          method: "POST",
          body: JSON.stringify({ full_name, phone_number: phone || undefined }),
        });
        toast("Пользователь создан");
      }
      invalidateDirectory();
      closeModal();
      loadUsers();
    } catch (err) {
      if (err.status === 409 || err.status === 404) closeModal();
      handleMutationError(err, loadUsers);
    }
  };
}
async function deleteUser(id) {
  if (!confirm("Удалить пользователя?")) return;
  try {
    await api(
      `/users/${id}`,
      { method: "DELETE" },
      { 409: "У пользователя есть задачи — сначала удалите или завершите их." },
    );
    toast("Пользователь удалён");
    invalidateDirectory();
    if (state.users.length === 1 && state.userPage > 0) state.userPage--;
    loadUsers();
  } catch (e) {
    toast(e.message, true);
    if (e.status === 404) loadUsers();
  }
}

// ---------- Статистика ----------
// API принимает `to` не включительно, а в интерфейсе «по» — включительно.
const nextDay = date => {
  const d = new Date(`${date}T00:00:00Z`);
  d.setUTCDate(d.getUTCDate() + 1);
  return d.toISOString().slice(0, 10);
};
async function loadStats() {
  renderShell("Статистика", "Сводная аналитика по задачам");
  renderLoader();
  const f = state.statsFilter;
  const q = new URLSearchParams();
  if (f.user) q.set("user_id", f.user);
  if (f.from) q.set("from", f.from);
  if (f.to) q.set("to", nextDay(f.to));
  try {
    const [s] = await Promise.all([
      api(`/statistics?${q.toString()}`),
      loadDirectory(),
    ]);
    state.stats = s;
    renderStats(s);
  } catch (e) {
    renderError(e);
  }
}
function bar(label, count, total) {
  const pct = total ? (count / total) * 100 : 0;
  return `<div class="bar-col">
    <div class="bar-value">${count} · ${Math.round(pct)}%</div>
    <div class="bar" style="height:${pct}%" role="img" aria-label="${label}: ${count} из ${total}"></div>
    <div class="bar-label">${label}</div>
  </div>`;
}
function renderStats(s) {
  const f = state.statsFilter;
  const created = Number(s?.tasks_created || 0),
    completed = Number(s?.tasks_completed || 0),
    rate = Math.round(Number(s?.tasks_completed_rate || 0) * 10) / 10,
    avg = esc(s?.tasks_average_completion_time || "—");
  const userOptions = state.directory
    .map(
      u =>
        `<option value="${u.id}" ${String(u.id) === f.user ? "selected" : ""}>${esc(u.full_name)}</option>`,
    )
    .join("");
  $("#page-content").innerHTML = `
    <div class="filters stats">
      <label class="field-label" for="statUser">Пользователь</label><select class="control" id="statUser"><option value="">Все</option>${userOptions}</select>
      <label class="field-label" for="statFrom">С</label><input class="control" type="date" id="statFrom" value="${esc(f.from)}">
      <label class="field-label" for="statTo">По</label><input class="control" type="date" id="statTo" value="${esc(f.to)}">
      <div class="filter-actions"><button class="btn reset" id="applyStats">Применить</button><button class="btn reset" id="resetStats">Сбросить</button></div>
    </div>
    <div class="stats-grid">
      <div class="stat-card"><div class="stat-label">Создано задач</div><div class="stat-value">${created}</div></div>
      <div class="stat-card"><div class="stat-label">Выполнено</div><div class="stat-value">${completed}</div></div>
      <div class="stat-card"><div class="stat-label">Процент выполнения</div><div class="stat-value">${rate}%</div></div>
      <div class="stat-card"><div class="stat-label">Среднее время выполнения</div><div class="stat-value" style="font-size:25px">${avg}</div></div>
    </div>
    <div class="bar-wrap">
      <div style="font-size:17px;font-weight:700;margin-bottom:14px">Состояние задач</div>
      ${created ? `<div class="bars">${bar("Открытые", created - completed, created)}${bar("Выполненные", completed, created)}</div>` : `<div class="subtitle">За выбранный период задач нет</div>`}
    </div>`;
  $("#applyStats").onclick = () => {
    const from = $("#statFrom").value;
    const to = $("#statTo").value;
    if (from && to && to < from) {
      toast("Дата «по» не может быть раньше даты «с».", true);
      return;
    }
    state.statsFilter = { user: $("#statUser").value, from, to };
    loadStats();
  };
  $("#resetStats").onclick = () => {
    state.statsFilter = { user: "", from: "", to: "" };
    loadStats();
  };
}

// ---------- Модальное окно ----------
let lastFocused = null;
const isModalOpen = () => $("#modalBackdrop").classList.contains("show");
function showModal() {
  lastFocused = document.activeElement;
  $("#modalBackdrop").classList.add("show");
  document.body.style.overflow = "hidden";
  $("#closeModal").onclick = closeModal;
  $("#cancelModal").onclick = closeModal;
  $("#modal").querySelector(".control")?.focus();
}
function closeModal() {
  if (!isModalOpen()) return;
  $("#modalBackdrop").classList.remove("show");
  document.body.style.overflow = "";
  if (lastFocused && document.contains(lastFocused)) lastFocused.focus();
}
$("#modalBackdrop").addEventListener("click", e => {
  if (e.target.id === "modalBackdrop") closeModal();
});
document.addEventListener("keydown", e => {
  if (!isModalOpen()) return;
  if (e.key === "Escape") {
    closeModal();
    return;
  }
  // Не выпускаем фокус из открытого диалога.
  if (e.key === "Tab") {
    const focusable = [
      ...$("#modal").querySelectorAll("button, input, select, textarea"),
    ].filter(el => !el.disabled);
    if (!focusable.length) return;
    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    if (e.shiftKey && document.activeElement === first) {
      e.preventDefault();
      last.focus();
    } else if (!e.shiftKey && document.activeElement === last) {
      e.preventDefault();
      first.focus();
    }
  }
});

// ---------- Навигация и тема ----------
async function switchPage(page) {
  state.page = page;
  setActive();
  if (page === "tasks") return loadTasks();
  if (page === "users") return loadUsers();
  return loadStats();
}
document
  .querySelectorAll(".nav button")
  .forEach(b => (b.onclick = () => switchPage(b.dataset.page)));
let savedTheme = null;
try {
  savedTheme = localStorage.getItem("todo-theme");
} catch {}
const prefersDark = window.matchMedia?.("(prefers-color-scheme: dark)").matches;
if (savedTheme === "dark" || (!savedTheme && prefersDark))
  document.body.classList.add("dark");
$("#themeSwitch").onclick = () => {
  document.body.classList.toggle("dark");
  try {
    localStorage.setItem(
      "todo-theme",
      document.body.classList.contains("dark") ? "dark" : "light",
    );
  } catch {}
};
setActive();
loadTasks();
