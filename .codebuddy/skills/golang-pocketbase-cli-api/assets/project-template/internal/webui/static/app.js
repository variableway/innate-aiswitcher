const API = '/api/myapp';

async function fetchJSON(url, opts) {
  const res = await fetch(url, opts);
  const data = await res.json();
  if (!res.ok) throw new Error(data.message || 'Request failed');
  return data;
}

async function loadItems() {
  try {
    const items = await fetchJSON(`${API}/items`);
    const tbody = document.getElementById('items-body');
    tbody.innerHTML = items.map(i =>
      `<tr><td>${i.slug}</td><td>${i.name}</td><td>${i.active ? '✓' : '✗'}</td></tr>`
    ).join('');
  } catch (err) {
    console.error('Failed to load items:', err);
  }
}

document.getElementById('form').addEventListener('submit', async (e) => {
  e.preventDefault();
  const form = e.target;
  const data = {
    slug: form.slug.value,
    name: form.name.value,
    description: form.description.value,
  };
  try {
    await fetchJSON(`${API}/items`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    form.reset();
    loadItems();
  } catch (err) {
    alert('Error: ' + err.message);
  }
});

loadItems();
