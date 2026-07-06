// AISwitcher Configuration Web UI
const API = '/api/aisw';

// === State ===
let providers = [];
let profiles = [];
let agents = [];
let presets = [];
let confirmCallback = null;

// === Init ===
document.addEventListener('DOMContentLoaded', () => {
  initTabs();
  initModals();
  loadProviders();
  loadProfiles();
  loadAgents();
  loadPresets();
  initForms();
});

// === Tabs ===
function initTabs() {
  document.querySelectorAll('.tab').forEach(tab => {
    tab.addEventListener('click', () => {
      document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
      document.querySelectorAll('.panel').forEach(p => p.classList.remove('active'));
      tab.classList.add('active');
      document.getElementById(tab.dataset.tab + '-panel').classList.add('active');
    });
  });
}

// === Modals ===
function initModals() {
  document.querySelectorAll('.modal-close').forEach(btn => {
    btn.addEventListener('click', () => closeAllModals());
  });
  document.querySelectorAll('.modal-cancel').forEach(btn => {
    btn.addEventListener('click', () => closeAllModals());
  });
  document.querySelectorAll('.modal').forEach(modal => {
    modal.addEventListener('click', (e) => {
      if (e.target === modal) closeAllModals();
    });
  });
}

function openModal(id) { document.getElementById(id).classList.add('open'); }
function closeAllModals() {
  document.querySelectorAll('.modal').forEach(m => m.classList.remove('open'));
}

// === Toast ===
function toast(message, type = 'info') {
  const container = document.getElementById('toast-container');
  const el = document.createElement('div');
  el.className = `toast toast-${type}`;
  el.textContent = message;
  container.appendChild(el);
  setTimeout(() => {
    el.style.animation = 'slideOut 0.2s ease-in forwards';
    setTimeout(() => el.remove(), 200);
  }, 3000);
}

// === API Helpers ===
async function apiGet(path) {
  const res = await fetch(API + path);
  if (!res.ok) throw new Error((await res.json().catch(() => ({}))).message || res.statusText);
  return res.json();
}

async function apiPost(path, body) {
  const res = await fetch(API + path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  const data = await res.json();
  if (!res.ok) throw new Error(data.message || res.statusText);
  return data;
}

async function apiPut(path, body) {
  const res = await fetch(API + path, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  const data = await res.json();
  if (!res.ok) throw new Error(data.message || res.statusText);
  return data;
}

async function apiDelete(path) {
  const res = await fetch(API + path, { method: 'DELETE' });
  const data = await res.json();
  if (!res.ok) throw new Error(data.message || res.statusText);
  return data;
}

// === Load Data ===
async function loadProviders() {
  try {
    providers = await apiGet('/providers');
    renderProviders();
  } catch (e) {
    document.getElementById('providers-list').innerHTML = `<div class="loading">Failed to load: ${e.message}</div>`;
  }
}

async function loadProfiles() {
  try {
    profiles = await apiGet('/profiles');
    renderProfiles();
  } catch (e) {
    document.getElementById('profiles-list').innerHTML = `<div class="loading">Failed to load: ${e.message}</div>`;
  }
}

async function loadAgents() {
  try {
    agents = await apiGet('/agents');
    populateAgentSelect();
  } catch (e) {
    console.error('Failed to load agents:', e);
  }
}

async function loadPresets() {
  try {
    presets = await apiGet('/presets');
  } catch (e) {
    console.error('Failed to load presets:', e);
  }
}

// === Render Providers ===
function renderProviders() {
  const container = document.getElementById('providers-list');
  if (providers.length === 0) {
    container.innerHTML = '<div class="loading">No providers configured. Click "+ Add Provider" to get started.</div>';
    return;
  }
  container.innerHTML = providers.map(p => {
    const protocolLabels = { openai_chat: 'OpenAI Chat', anthropic: 'Anthropic', openai_responses: 'OpenAI Responses', gemini_native: 'Gemini', generic: 'Generic' };
    const hasKey = p.api_key && p.api_key !== '';
    return `
    <div class="card">
      <div class="card-icon">🔌</div>
      <div class="card-body">
        <div class="card-title">${esc(p.name)} <span class="badge ${p.active ? 'badge-active' : 'badge-inactive'}">${p.active ? 'active' : 'inactive'}</span> <span class="badge ${hasKey ? 'badge-key-set' : 'badge-key-missing'}">${hasKey ? 'key set' : 'no key'}</span></div>
        <div class="card-meta">
          <span>${esc(p.slug)}</span>
          <span>${protocolLabels[p.api_protocol] || p.api_protocol}</span>
          ${p.default_model ? `<span>Model: ${esc(p.default_model)}</span>` : ''}
          <span title="${esc(p.base_url)}">${truncate(p.base_url, 40)}</span>
        </div>
      </div>
      <div class="card-actions">
        <button class="btn btn-secondary btn-sm" onclick="testProvider('${esc(p.slug)}')" title="Test connection">Test</button>
        <button class="btn btn-secondary btn-sm" onclick="editProvider('${esc(p.slug)}')">Edit</button>
        <button class="btn btn-danger btn-sm" onclick="deleteProvider('${esc(p.slug)}', '${esc(p.name)}')">Delete</button>
      </div>
    </div>`;
  }).join('');
}

function renderProfiles() {
  const container = document.getElementById('profiles-list');
  if (profiles.length === 0) {
    container.innerHTML = '<div class="loading">No profiles configured. Click "+ Add Profile" to create one.</div>';
    return;
  }
  container.innerHTML = profiles.map(p => `
    <div class="card">
      <div class="card-icon">📋</div>
      <div class="card-body">
        <div class="card-title">${esc(p.name)} ${p.is_default ? '<span class="badge badge-active">default</span>' : ''}</div>
        <div class="card-meta">
          <span>${esc(p.slug)}</span>
          <span>Agent: ${esc(p.agent)}</span>
          <span>Provider: ${esc(p.provider)}</span>
          ${p.model ? `<span>Model: ${esc(p.model)}</span>` : ''}
        </div>
      </div>
      <div class="card-actions">
        <button class="btn btn-secondary btn-sm" onclick="editProfile('${esc(p.slug)}')">Edit</button>
        <button class="btn btn-danger btn-sm" onclick="deleteProfile('${esc(p.slug)}', '${esc(p.name)}')">Delete</button>
      </div>
    </div>
  `).join('');
}

// === Provider CRUD ===
function populateProviderSelect(slug) {
  const sel = document.getElementById('prform-provider');
  sel.innerHTML = '<option value="">Select provider...</option>' +
    providers.map(p => `<option value="${esc(p.slug)}">${esc(p.name)} (${esc(p.slug)})</option>`).join('');
  if (slug) sel.value = slug;
}

function populateAgentSelect(slug) {
  const sel = document.getElementById('prform-agent');
  sel.innerHTML = '<option value="">Select agent...</option>' +
    agents.map(a => `<option value="${esc(a.slug)}">${esc(a.name)} (${esc(a.slug)})</option>`).join('');
  if (slug) sel.value = slug;
}

document.getElementById('btn-add-provider').addEventListener('click', () => {
  document.getElementById('provider-modal-title').textContent = 'Add Provider';
  document.getElementById('provider-form').reset();
  document.getElementById('pform-slug-orig').value = '';
  document.getElementById('pform-api-key').type = 'password';
  openModal('provider-modal');
});

document.getElementById('btn-add-from-preset').addEventListener('click', () => {
  renderPresetList();
  document.getElementById('preset-list').style.display = 'block';
  document.getElementById('preset-options').style.display = 'none';
  openModal('preset-modal');
});

function editProvider(slug) {
  const p = providers.find(x => x.slug === slug);
  if (!p) return;
  document.getElementById('provider-modal-title').textContent = 'Edit Provider';
  document.getElementById('pform-slug-orig').value = p.slug;
  document.getElementById('pform-slug').value = p.slug;
  document.getElementById('pform-name').value = p.name;
  document.getElementById('pform-base-url').value = p.base_url;
  document.getElementById('pform-api-key').value = p.api_key || '';
  document.getElementById('pform-api-key').type = 'password';
  document.getElementById('pform-protocol').value = p.api_protocol || 'openai_chat';
  document.getElementById('pform-model').value = p.default_model || '';
  document.getElementById('pform-notes').value = p.notes || '';
  document.getElementById('pform-active').checked = p.active;
  openModal('provider-modal');
}

function deleteProvider(slug, name) {
  confirmCallback = async () => {
    try {
      await apiDelete('/providers/' + slug);
      toast(`Provider "${name}" deleted`, 'success');
      await loadProviders();
      await loadProfiles();
    } catch (e) {
      toast('Delete failed: ' + e.message, 'error');
    }
  };
  document.getElementById('confirm-message').textContent = `Are you sure you want to delete provider "${name}"? This cannot be undone.`;
  openModal('confirm-modal');
}

// === Profile CRUD ===
document.getElementById('btn-add-profile').addEventListener('click', () => {
  document.getElementById('profile-modal-title').textContent = 'Add Profile';
  document.getElementById('profile-form').reset();
  document.getElementById('prform-slug-orig').value = '';
  populateAgentSelect();
  populateProviderSelect();
  openModal('profile-modal');
});

function editProfile(slug) {
  const p = profiles.find(x => x.slug === slug);
  if (!p) return;
  document.getElementById('profile-modal-title').textContent = 'Edit Profile';
  document.getElementById('prform-slug-orig').value = p.slug;
  document.getElementById('prform-slug').value = p.slug;
  document.getElementById('prform-name').value = p.name;
  populateAgentSelect(p.agent);
  populateProviderSelect(p.provider);
  document.getElementById('prform-model').value = p.model || '';
  document.getElementById('prform-args').value = p.default_args || '';
  document.getElementById('prform-skip').value = p.skip_permissions || '';
  document.getElementById('prform-default').checked = p.is_default;
  openModal('profile-modal');
}

function deleteProfile(slug, name) {
  confirmCallback = async () => {
    try {
      await apiDelete('/profiles/' + slug);
      toast(`Profile "${name}" deleted`, 'success');
      await loadProfiles();
    } catch (e) {
      toast('Delete failed: ' + e.message, 'error');
    }
  };
  document.getElementById('confirm-message').textContent = `Are you sure you want to delete profile "${name}"?`;
  openModal('confirm-modal');
}

// === Forms ===
function initForms() {
  // Provider form
  document.getElementById('provider-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const slugOrig = document.getElementById('pform-slug-orig').value;
    const data = {
      slug: document.getElementById('pform-slug').value.trim(),
      name: document.getElementById('pform-name').value.trim(),
      base_url: document.getElementById('pform-base-url').value.trim(),
      api_key: document.getElementById('pform-api-key').value,
      api_protocol: document.getElementById('pform-protocol').value,
      default_model: document.getElementById('pform-model').value.trim(),
      notes: document.getElementById('pform-notes').value.trim(),
      active: document.getElementById('pform-active').checked,
    };
    try {
      if (slugOrig) {
        await apiPut('/providers/' + slugOrig, data);
        toast('Provider updated', 'success');
      } else {
        await apiPost('/providers', data);
        toast('Provider created', 'success');
      }
      closeAllModals();
      await loadProviders();
      await loadProfiles();
    } catch (err) {
      toast(err.message, 'error');
    }
  });

  // Profile form
  document.getElementById('profile-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const slugOrig = document.getElementById('prform-slug-orig').value;
    const data = {
      slug: document.getElementById('prform-slug').value.trim(),
      name: document.getElementById('prform-name').value.trim(),
      agent: document.getElementById('prform-agent').value,
      provider: document.getElementById('prform-provider').value,
      model: document.getElementById('prform-model').value.trim(),
      default_args: document.getElementById('prform-args').value.trim(),
      skip_permissions: document.getElementById('prform-skip').value,
      is_default: document.getElementById('prform-default').checked,
    };
    try {
      if (slugOrig) {
        await apiPut('/profiles/' + slugOrig, data);
        toast('Profile updated', 'success');
      } else {
        await apiPost('/profiles', data);
        toast('Profile created', 'success');
      }
      closeAllModals();
      await loadProfiles();
    } catch (err) {
      toast(err.message, 'error');
    }
  });

  // Confirm delete
  document.getElementById('btn-confirm-delete').addEventListener('click', async () => {
    if (confirmCallback) {
      await confirmCallback();
      confirmCallback = null;
    }
    closeAllModals();
  });

  // Toggle password visibility
  document.querySelectorAll('.btn-toggle-password').forEach(btn => {
    btn.addEventListener('click', () => {
      const input = btn.previousElementSibling;
      if (input.type === 'password') {
        input.type = 'text';
        btn.textContent = '🙈';
      } else {
        input.type = 'password';
        btn.textContent = '👁';
      }
    });
  });
}

// === Preset Import ===
let selectedPreset = null;
let selectedOption = null;

function renderPresetList() {
  const container = document.getElementById('preset-list');
  if (presets.length === 0) {
    container.innerHTML = '<div class="loading">No presets available.</div>';
    return;
  }
  container.innerHTML = presets.map(p => `
    <div class="preset-item" onclick="selectPreset('${esc(p.slug)}')">
      <div class="preset-item-name">${esc(p.name)}</div>
      <div class="preset-item-opts">${p.url_options ? p.url_options.map(o => o.label).join(', ') : ''}</div>
    </div>
  `).join('');
}

function selectPreset(slug) {
  selectedPreset = presets.find(p => p.slug === slug);
  selectedOption = null;
  if (!selectedPreset) return;

  document.getElementById('preset-list').style.display = 'none';
  document.getElementById('preset-options').style.display = 'block';

  const optionList = document.getElementById('preset-option-list');
  if (selectedPreset.url_options.length === 1) {
    selectedOption = selectedPreset.url_options[0];
    optionList.innerHTML = `<div class="preset-option-item selected">
      <div class="opt-label">${esc(selectedOption.label)}</div>
      <div class="opt-detail">${esc(selectedOption.api_protocol)} — ${esc(selectedOption.base_url)}</div>
    </div>`;
  } else {
    optionList.innerHTML = selectedPreset.url_options.map(o => `
      <div class="preset-option-item" onclick="selectOption('${esc(o.slug)}')" id="opt-${esc(o.slug)}">
        <div class="opt-label">${esc(o.label)}</div>
        <div class="opt-detail">${esc(o.api_protocol)} — ${esc(o.base_url)}</div>
      </div>
    `).join('');
  }
}

function selectOption(slug) {
  if (!selectedPreset) return;
  selectedOption = selectedPreset.url_options.find(o => o.slug === slug);
  document.querySelectorAll('.preset-option-item').forEach(el => el.classList.remove('selected'));
  const el = document.getElementById('opt-' + slug);
  if (el) el.classList.add('selected');
}

document.getElementById('btn-preset-back').addEventListener('click', () => {
  document.getElementById('preset-list').style.display = 'block';
  document.getElementById('preset-options').style.display = 'none';
  selectedPreset = null;
  selectedOption = null;
});

document.getElementById('btn-preset-import').addEventListener('click', async () => {
  if (!selectedPreset || !selectedOption) {
    toast('Please select a provider and option.', 'info');
    return;
  }
  try {
    await apiPost('/providers/from-preset', {
      preset_slug: selectedPreset.slug,
      option_slug: selectedOption.slug,
      api_key: document.getElementById('preset-api-key').value,
    });
    toast(`Provider "${selectedPreset.name}" imported`, 'success');
    closeAllModals();
    await loadProviders();
    await loadProfiles();
  } catch (err) {
    toast('Import failed: ' + err.message, 'error');
  }
});

// === Test Provider ===
async function testProvider(slug) {
  openModal('test-modal');
  const resultDiv = document.getElementById('test-result');
  resultDiv.innerHTML = '<div class="loading">Testing connection...</div>';

  try {
    const data = await apiPost('/providers/' + slug + '/test', {});
    resultDiv.innerHTML = data.ok
      ? `<div class="test-success">
           <strong>✓ Connection OK</strong>
           <pre>${esc(JSON.stringify(data, null, 2))}</pre>
         </div>`
      : `<div class="test-error">
           <strong>✗ Connection Failed</strong>
           <pre>${esc(JSON.stringify(data, null, 2))}</pre>
         </div>`;
  } catch (e) {
    resultDiv.innerHTML = `<div class="test-error"><strong>✗ Error</strong><pre>${esc(e.message)}</pre></div>`;
  }
}

// === Helpers ===
function esc(s) {
  if (!s) return '';
  return String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;').replace(/'/g, '&#39;');
}

function truncate(s, len) {
  if (!s) return '';
  return s.length > len ? s.slice(0, len) + '...' : s;
}
