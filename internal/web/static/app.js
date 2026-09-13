'use strict';

// ---------------------------------------------------------------------------
// State
// ---------------------------------------------------------------------------

const SIZES = ['original', '12x12', '25x25'];
const TERMINAL = ['complete', 'failed', 'rejected', 'abandoned'];

// Served by httpserver: same origin. Served by `make web` on :5500: API is on :8080.
function defaultBase() {
  return location.port === '5500' ? 'http://localhost:8080' : location.origin;
}

const store = {
  get base() { return localStorage.getItem('gis:base') || defaultBase(); },
  set base(v) { localStorage.setItem('gis:base', v); },
  get token() { return localStorage.getItem('gis:token') || ''; },
  set token(v) { v ? localStorage.setItem('gis:token', v) : localStorage.removeItem('gis:token'); },
};

const $ = (sel) => document.querySelector(sel);
const el = (tag, cls, text) => {
  const n = document.createElement(tag);
  if (cls) n.className = cls;
  if (text !== undefined) n.textContent = text;
  return n;
};

// Blob URLs created for variant previews; revoked before each new render so the
// page does not leak object URLs as the user clicks through images.
let variantURLs = [];

// ---------------------------------------------------------------------------
// JWT
// ---------------------------------------------------------------------------

// decodeJWT reads the payload without verifying the signature. Only the server
// can verify it — this is purely to show what the token carries.
function decodeJWT(token) {
  try {
    const payload = token.split('.')[1];
    const json = atob(payload.replace(/-/g, '+').replace(/_/g, '/'));
    return JSON.parse(json);
  } catch {
    return null;
  }
}

function currentUserID() {
  const claims = decodeJWT(store.token);
  return claims ? claims.sub : '';
}

// ---------------------------------------------------------------------------
// HTTP
// ---------------------------------------------------------------------------

// api calls the backend and records every request in the log table. It returns
// { ok, status, data } where data is parsed JSON for success responses and the
// plain-text message for errors — the server sends errors via http.Error, which
// is text/plain, so calling .json() on a failure would throw.
async function api(method, path, { body, raw = false } = {}) {
  const url = store.base.replace(/\/+$/, '') + path;
  const headers = {};
  if (store.token) headers['Authorization'] = 'Bearer ' + store.token;
  if (body !== undefined) headers['Content-Type'] = 'application/json';

  const started = performance.now();
  let res;
  try {
    res = await fetch(url, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
    });
  } catch (err) {
    logRequest(method, url, 0, performance.now() - started);
    return { ok: false, status: 0, data: 'network error — is the API running at ' + store.base + '?' };
  }
  logRequest(method, url, res.status, performance.now() - started);

  if (!res.ok) return { ok: false, status: res.status, data: (await res.text()).trim() };
  if (raw) return { ok: true, status: res.status, data: await res.blob() };
  if (res.status === 204 || res.headers.get('content-length') === '0') {
    return { ok: true, status: res.status, data: null };
  }
  const text = await res.text();
  return { ok: true, status: res.status, data: text ? JSON.parse(text) : null };
}

function logRequest(method, url, status, ms) {
  const row = el('tr');
  row.append(el('td', null, method));
  row.append(el('td', null, url.replace(store.base, '')));
  row.append(el('td', 'status-' + String(status).charAt(0), status || 'ERR'));
  row.append(el('td', null, Math.round(ms) + 'ms'));
  $('#log tbody').prepend(row);
}

// ---------------------------------------------------------------------------
// Session
// ---------------------------------------------------------------------------

function setOutput(node, message, ok) {
  node.textContent = message;
  node.className = ok ? 'ok' : 'err';
}

function applySession() {
  const token = store.token;
  const claims = token ? decodeJWT(token) : null;
  const signedIn = Boolean(claims);

  $('#whoami').classList.toggle('hidden', !signedIn);
  $('#token-box').classList.toggle('hidden', !signedIn);
  for (const id of ['#upload-section', '#images-section', '#authz-section']) {
    $(id).classList.toggle('locked', !signedIn);
  }
  if (!signedIn) {
    $('#variants-section').classList.add('hidden');
    $('#image-list').innerHTML = '';
    return;
  }

  $('#whoami-email').textContent = localStorage.getItem('gis:email') || claims.sub;
  $('#token-raw').textContent = token;
  $('#token-claims').textContent = JSON.stringify(
    {
      sub: claims.sub,
      iat: claims.iat ? new Date(claims.iat * 1000).toISOString() : null,
      exp: claims.exp ? new Date(claims.exp * 1000).toISOString() : null,
    },
    null,
    2,
  );
  refreshImages();
}

// ---------------------------------------------------------------------------
// 1 · Auth
// ---------------------------------------------------------------------------

$('#register-form').addEventListener('submit', async (e) => {
  e.preventDefault();
  const out = $('#register-out');
  const { email, password } = Object.fromEntries(new FormData(e.target));
  const res = await api('POST', '/register', { body: { email, password } });

  if (!res.ok) {
    // The auth service collapses a duplicate-email unique-constraint violation
    // into codes.Internal, so a 500 here most often means the email is taken.
    const extra = res.status === 500 ? ' (this email may already be registered)' : '';
    setOutput(out, res.status + ' — ' + res.data + extra, false);
    return;
  }
  setOutput(out, 'created user ' + res.data.id + ' — now log in', true);
  $('#login-form').elements.email.value = email;
  $('#login-form').elements.password.value = password;
});

$('#login-form').addEventListener('submit', async (e) => {
  e.preventDefault();
  const out = $('#login-out');
  const { email, password } = Object.fromEntries(new FormData(e.target));
  const res = await api('POST', '/login', { body: { email, password } });

  if (!res.ok) {
    setOutput(out, res.status + ' — ' + res.data, false);
    return;
  }
  store.token = res.data.token;
  localStorage.setItem('gis:email', email);
  setOutput(out, 'logged in', true);
  e.target.reset();
  applySession();
});

// Switching accounts is the whole point of the authorization demo, so logging
// out clears both forms rather than leaving the previous user's email behind.
$('#logout').addEventListener('click', () => {
  store.token = '';
  localStorage.removeItem('gis:email');
  $('#register-form').reset();
  $('#login-form').reset();
  $('#login-out').textContent = '';
  $('#register-out').textContent = '';
  $('#steps').classList.add('hidden');
  applySession();
});

// ---------------------------------------------------------------------------
// 2 · Upload
// ---------------------------------------------------------------------------

function step(name, state, detail) {
  const li = document.querySelector(`.steps li[data-step="${name}"]`);
  li.className = state;
  li.querySelector('.detail').textContent = detail ? '— ' + detail : '';
}

$('#upload-form').addEventListener('submit', async (e) => {
  e.preventDefault();
  const file = $('#file').files[0];
  if (!file) return;

  const button = e.target.querySelector('button');
  button.disabled = true;
  $('#steps').classList.remove('hidden');
  for (const name of ['negotiate', 'put', 'complete', 'poll']) step(name, '', '');

  try {
    // Step 1 — negotiate. The API records a pending row, starts the Temporal
    // workflow, and returns a presigned PUT URL.
    step('negotiate', 'active');
    const neg = await api('POST', '/uploads', {
      body: { filename: file.name, content_type: file.type },
    });
    if (!neg.ok) {
      step('negotiate', 'failed', neg.status + ' ' + neg.data);
      return;
    }
    const { id, expires_in, url } = neg.data;
    step('negotiate', 'done', `id ${id}, url expires in ${expires_in}s`);
    rememberID(id, file.name);

    // Step 2 — the bytes go straight to object storage, bypassing the API.
    step('put', 'active');
    const started = performance.now();
    const put = await fetch(url, { method: 'PUT', body: file });
    logRequest('PUT', '(presigned) ' + new URL(url).pathname, put.status, performance.now() - started);
    if (!put.ok) {
      step('put', 'failed', 'storage returned ' + put.status);
      return;
    }
    step('put', 'done', `${(file.size / 1024).toFixed(0)} KB uploaded directly to storage`);

    // Step 3 — signal the workflow, which is waiting on this or a 15m timeout.
    step('complete', 'active');
    const done = await api('POST', `/uploads/${id}/complete`);
    if (!done.ok) {
      step('complete', 'failed', done.status + ' ' + done.data);
      return;
    }
    step('complete', 'done', 'workflow signalled');

    // Step 4 — poll until the workflow reaches a terminal status.
    step('poll', 'active', 'waiting for the resize workflow…');
    const status = await pollStatus(id);
    step('poll', status === 'complete' ? 'done' : 'failed', 'status: ' + status);

    await refreshImages();
    if (status === 'complete') showVariants(id, file.name);
  } finally {
    button.disabled = false;
    $('#upload-form').reset();
  }
});

// pollStatus polls GET /images/{id}/status until the workflow settles. The
// pipeline moves pending → validating → processing → complete, so anything not
// in TERMINAL means work is still in flight.
async function pollStatus(id, timeoutMs = 90000) {
  const deadline = Date.now() + timeoutMs;
  let last = 'pending';
  while (Date.now() < deadline) {
    const res = await api('GET', `/images/${id}/status`);
    if (!res.ok) return 'error: ' + res.data;
    last = res.data.status;
    step('poll', 'active', 'status: ' + last);
    if (TERMINAL.includes(last)) return last;
    await new Promise((r) => setTimeout(r, 1000));
  }
  return 'timed out at ' + last;
}

// ---------------------------------------------------------------------------
// 3 · My images
// ---------------------------------------------------------------------------

// The ids this browser uploaded, keyed by user id. Used only as a fallback for
// the not-yet-implemented GET /images; the server is the real source of truth.
function localKey() { return 'gis:ids:' + currentUserID(); }
function localIDs() { return JSON.parse(localStorage.getItem(localKey()) || '[]'); }
function rememberID(id, filename) {
  const ids = localIDs().filter((r) => r.id !== id);
  ids.unshift({ id, filename });
  localStorage.setItem(localKey(), JSON.stringify(ids));
}

$('#refresh').addEventListener('click', () => refreshImages());

async function refreshImages() {
  if (!store.token) return;
  const res = await api('GET', '/images');

  if (res.ok && res.data && Array.isArray(res.data.images)) {
    $('#list-fallback').classList.add('hidden');
    renderImages(res.data.images);
    return;
  }

  // GET /images is not implemented yet. Fall back to the ids this browser
  // uploaded so the demo still runs, and say so plainly — this list is NOT
  // server-filtered, so it does not prove per-user isolation on its own.
  $('#list-fallback').classList.remove('hidden');
  $('#list-fallback').textContent =
    `GET /images returned ${res.status}. Falling back to ids remembered in this browser — ` +
    `these are not filtered by the server. Implement the endpoint to demo real per-user isolation.`;

  const rows = [];
  for (const { id, filename } of localIDs()) {
    const st = await api('GET', `/images/${id}/status`);
    // A 404 here means the row is gone or belongs to another account.
    if (st.ok) rows.push({ id, filename, status: st.data.status });
  }
  renderImages(rows);
}

function renderImages(images) {
  const list = $('#image-list');
  list.innerHTML = '';
  if (!images.length) {
    list.append(el('p', 'empty', 'No images for this account yet.'));
    return;
  }
  for (const img of images) {
    const name = img.filename || img.original_filename || '(unnamed)';
    const card = el('div', 'card');
    card.append(el('div', 'badge ' + img.status, img.status));
    card.append(el('div', 'name', name));
    card.append(el('div', 'id', img.id));
    card.addEventListener('click', () => showVariants(img.id, name, img.status));
    list.append(card);
  }
}

// ---------------------------------------------------------------------------
// 4 · Variants
// ---------------------------------------------------------------------------

// showVariants fetches each size with the bearer token and renders it from a
// blob URL. A plain <img src="/images/{id}?size=..."> cannot work: the browser
// will not attach an Authorization header to an image request, so it would 401.
async function showVariants(id, filename, status) {
  const box = $('#variants');
  $('#variants-section').classList.remove('hidden');
  variantURLs.forEach(URL.revokeObjectURL);
  variantURLs = [];
  box.innerHTML = '';

  const heading = $('#variants-section').querySelector('h2');
  heading.textContent = `4 · Resized variants — ${filename}`;

  if (status && status !== 'complete') {
    box.append(el('p', 'empty', `This upload is "${status}". Variants exist only once it is complete.`));
    return;
  }

  for (const size of SIZES) {
    const res = await api('GET', `/images/${id}?size=${size}`, { raw: true });
    const card = el('div', 'variant');
    card.append(el('h4', null, size));

    if (!res.ok) {
      card.append(el('p', 'empty', res.status + ' — not available'));
      box.append(card);
      continue;
    }

    const url = URL.createObjectURL(res.data);
    variantURLs.push(url);

    const image = el('img');
    if (size !== 'original') image.classList.add('pixelated');
    image.src = url;

    const meta = el('div', 'meta', '…');
    image.addEventListener('load', () => {
      meta.textContent = `${image.naturalWidth}×${image.naturalHeight}px · ${(res.data.size / 1024).toFixed(1)} KB`;
    });

    card.append(image, meta);
    box.append(card);
  }
  $('#variants-section').scrollIntoView({ behavior: 'smooth', block: 'nearest' });
}

// ---------------------------------------------------------------------------
// 5 · Authorization probe
// ---------------------------------------------------------------------------

$('#probe-form').addEventListener('submit', async (e) => {
  e.preventDefault();
  const out = $('#probe-out');
  const id = new FormData(e.target).get('id').trim();
  const res = await api('GET', `/images/${id}/status`);

  if (res.ok) {
    setOutput(out, `200 — you own this image. status: ${res.data.status}`, true);
    return;
  }
  const why = {
    400: 'the id is not 32 characters',
    401: 'no valid token — log in first',
    404: 'either no such image, or it belongs to another user. The server returns the same 404 for both, so nothing leaks about which.',
  }[res.status];
  setOutput(out, `${res.status} — ${res.data}${why ? '\n\n' + why : ''}`, false);
});

// ---------------------------------------------------------------------------
// Boot
// ---------------------------------------------------------------------------

$('#clear-log').addEventListener('click', () => { $('#log tbody').innerHTML = ''; });

$('#api-base').value = store.base;
$('#api-base').addEventListener('change', (e) => {
  store.base = e.target.value.trim();
  applySession();
});

applySession();
