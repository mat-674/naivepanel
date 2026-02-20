// ============================
// Users Management Page
// ============================

function renderUsers() {
    return `
        <div class="page-header">
            <h2>Users</h2>
            <p>Manage NaiveProxy client accounts</p>
        </div>
        <div class="page-body">
            <div class="actions-bar">
                <h3 id="users-count"></h3>
                <button class="btn btn-primary btn-sm" onclick="showCreateUserModal()">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 5v14M5 12h14"/></svg>
                    Add User
                </button>
            </div>
            <div id="users-table">
                <div class="loading"><div class="spinner"></div></div>
            </div>
        </div>
        <div id="user-modal-container"></div>
    `;
}

async function initUsers() {
    await loadUsers();
}

async function loadUsers() {
    try {
        const res = await API.getUsers();
        const users = res.data || [];

        const countEl = document.getElementById('users-count');
        if (countEl) countEl.textContent = `${users.length} user${users.length !== 1 ? 's' : ''}`;

        const tableDiv = document.getElementById('users-table');
        if (!tableDiv) return;

        if (users.length === 0) {
            tableDiv.innerHTML = `
                <div class="empty-state">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17 21v-2a4 4 0 00-4-4H5a4 4 0 00-4-4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 00-3-3.87"/><path d="M16 3.13a4 4 0 010 7.75"/></svg>
                    <h3>No users yet</h3>
                    <p>Create your first NaiveProxy user to get started</p>
                    <button class="btn btn-primary btn-sm" onclick="showCreateUserModal()">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 5v14M5 12h14"/></svg>
                        Add User
                    </button>
                </div>
            `;
            return;
        }

        tableDiv.innerHTML = `
            <div class="table-wrapper">
                <table>
                    <thead>
                        <tr>
                            <th>Username</th>
                            <th>Password</th>
                            <th>Traffic ↑</th>
                            <th>Traffic ↓</th>
                            <th>Limit</th>
                            <th>Status</th>
                            <th>Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        ${users.map(u => `
                            <tr>
                                <td style="font-weight:600;">${escapeHtml(u.username)}</td>
                                <td>
                                    <code style="font-size:12px;color:var(--text-muted);" id="pwd-${u.id}">••••••••</code>
                                    <button class="btn btn-ghost btn-sm" onclick="togglePassword(${u.id}, '${escapeHtml(u.password)}')" title="Show/Hide">
                                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
                                    </button>
                                </td>
                                <td>${formatBytes(u.traffic_up)}</td>
                                <td>${formatBytes(u.traffic_down)}</td>
                                <td>${u.traffic_limit > 0 ? formatBytes(u.traffic_limit) : '∞'}</td>
                                <td>
                                    ${u.enabled
                ? '<span class="badge badge-success"><span class="badge-dot"></span> Active</span>'
                : '<span class="badge badge-danger"><span class="badge-dot"></span> Disabled</span>'
            }
                                </td>
                                <td>
                                    <div class="btn-group">
                                        <button class="btn btn-ghost btn-icon" onclick="showUserLink(${u.id})" title="Connection Link">
                                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10 13a5 5 0 007.54.54l3-3a5 5 0 00-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 00-7.54-.54l-3 3a5 5 0 007.07 7.07l1.71-1.71"/></svg>
                                        </button>
                                        <button class="btn btn-ghost btn-icon" onclick="toggleUserStatus(${u.id}, ${!u.enabled})" title="${u.enabled ? 'Disable' : 'Enable'}">
                                            ${u.enabled
                ? '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18.36 6.64a9 9 0 11-12.73 0M12 2v10"/></svg>'
                : '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 11.08V12a10 10 0 11-5.93-9.14"/><path d="M22 4L12 14.01l-3-3"/></svg>'
            }
                                        </button>
                                        <button class="btn btn-ghost btn-icon" onclick="deleteUser(${u.id}, '${escapeHtml(u.username)}')" title="Delete" style="color:var(--danger);">
                                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/></svg>
                                        </button>
                                    </div>
                                </td>
                            </tr>
                        `).join('')}
                    </tbody>
                </table>
            </div>
        `;
    } catch (error) {
        Toast.error(error.message);
    }
}

function togglePassword(userId, password) {
    const el = document.getElementById(`pwd-${userId}`);
    if (!el) return;
    if (el.textContent === '••••••••') {
        el.textContent = password;
        el.style.color = 'var(--text-primary)';
    } else {
        el.textContent = '••••••••';
        el.style.color = 'var(--text-muted)';
    }
}

function showCreateUserModal() {
    const container = document.getElementById('user-modal-container');
    if (!container) return;

    container.innerHTML = `
        <div class="modal-overlay" onclick="closeUserModal(event)">
            <div class="modal" onclick="event.stopPropagation()">
                <div class="modal-header">
                    <h3>Add New User</h3>
                    <button class="modal-close" onclick="closeUserModal()">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6L6 18M6 6l12 12"/></svg>
                    </button>
                </div>
                <form id="create-user-form">
                    <div class="form-group">
                        <label class="form-label">Username</label>
                        <input type="text" class="form-input" id="new-username" placeholder="Leave empty to auto-generate">
                        <div class="form-hint">Optional — random username will be generated if empty</div>
                    </div>
                    <div class="form-group">
                        <label class="form-label">Password</label>
                        <input type="text" class="form-input" id="new-password" placeholder="Leave empty to auto-generate">
                        <div class="form-hint">Optional — random password will be generated if empty</div>
                    </div>
                    <div class="form-group">
                        <label class="form-label">Traffic Limit (GB)</label>
                        <input type="number" class="form-input" id="new-traffic-limit" placeholder="0 = unlimited" value="0" min="0">
                        <div class="form-hint">Set to 0 for unlimited traffic</div>
                    </div>
                    <div class="modal-footer">
                        <button type="button" class="btn btn-outline" onclick="closeUserModal()">Cancel</button>
                        <button type="submit" class="btn btn-primary" id="create-user-btn">Create User</button>
                    </div>
                </form>
            </div>
        </div>
    `;

    document.getElementById('create-user-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        const btn = document.getElementById('create-user-btn');
        btn.disabled = true;

        const trafficGB = parseFloat(document.getElementById('new-traffic-limit').value) || 0;

        try {
            await API.createUser({
                username: document.getElementById('new-username').value.trim(),
                password: document.getElementById('new-password').value,
                traffic_limit: Math.floor(trafficGB * 1024 * 1024 * 1024),
            });
            Toast.success('User created successfully');
            closeUserModal();
            await loadUsers();
        } catch (error) {
            Toast.error(error.message);
            btn.disabled = false;
        }
    });
}

function closeUserModal(event) {
    if (event && event.target !== event.currentTarget) return;
    const container = document.getElementById('user-modal-container');
    if (container) container.innerHTML = '';
}

async function toggleUserStatus(id, enabled) {
    try {
        await API.updateUser(id, { enabled });
        Toast.success(`User ${enabled ? 'enabled' : 'disabled'}`);
        await loadUsers();
    } catch (error) {
        Toast.error(error.message);
    }
}

async function deleteUser(id, username) {
    if (!confirm(`Delete user "${username}"? This cannot be undone.`)) return;

    try {
        await API.deleteUser(id);
        Toast.success('User deleted');
        await loadUsers();
    } catch (error) {
        Toast.error(error.message);
    }
}

async function showUserLink(userId) {
    try {
        const res = await API.getUserLink(userId);
        const link = res.data;

        const container = document.getElementById('user-modal-container');
        if (!container) return;

        container.innerHTML = `
            <div class="modal-overlay" onclick="closeUserModal(event)">
                <div class="modal" onclick="event.stopPropagation()">
                    <div class="modal-header">
                        <h3>Connection Link</h3>
                        <button class="modal-close" onclick="closeUserModal()">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6L6 18M6 6l12 12"/></svg>
                        </button>
                    </div>
                    <div class="qr-container">
                        ${link.qr_code ? `<img src="data:image/png;base64,${link.qr_code}" alt="QR Code">` : ''}
                        <div class="qr-uri">${escapeHtml(link.uri)}</div>
                        <button class="btn btn-primary btn-sm" onclick="copyToClipboard('${escapeHtml(link.uri)}')">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1"/></svg>
                            Copy Link
                        </button>
                    </div>
                </div>
            </div>
        `;
    } catch (error) {
        Toast.error(error.message);
    }
}

function copyToClipboard(text) {
    navigator.clipboard.writeText(text).then(() => {
        Toast.success('Link copied to clipboard!');
    }).catch(() => {
        // Fallback
        const textarea = document.createElement('textarea');
        textarea.value = text;
        document.body.appendChild(textarea);
        textarea.select();
        document.execCommand('copy');
        document.body.removeChild(textarea);
        Toast.success('Link copied to clipboard!');
    });
}

function escapeHtml(str) {
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
}
