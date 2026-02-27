// ============================
// Users Management Page
// ============================

function renderUsers() {
    return `
        <div class="page-header">
            <h2>Users Management</h2>
            <p class="text-muted">Manage NaiveProxy client accounts</p>
        </div>
        <div class="glass-panel mt-4">
            <div class="flex justify-between align-center mb-4">
                <h3 id="users-count"></h3>
                <button class="btn btn-primary" onclick="showCreateUserModal()">Add User</button>
            </div>
            <div id="users-table">
                <div class="text-center text-muted">Loading...</div>
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

        let subPath = "sub";
        try {
            const setRes = await API.getSettings();
            if (setRes.data && setRes.data.sub_path) subPath = setRes.data.sub_path;
        } catch (e) { }

        const host = window.location.origin;

        tableDiv.innerHTML = `
            <div class="table-container">
                <table>
                    <thead>
                        <tr>
                            <th>Username</th>
                            <th>Traffic Limit</th>
                            <th>HWID Limit</th>
                            <th>Status</th>
                            <th>Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        ${users.map(u => `
                            <tr>
                                <td style="font-weight:600;">${escapeHtml(u.username)}</td>
                                <td>${u.traffic_limit > 0 ? formatBytes(u.traffic_limit) : '<span class="text-muted">∞</span>'}</td>
                                <td>${u.hwid_limit > 0 ? u.hwid_limit + ' devices' : '<span class="text-muted">Unlimited</span>'}</td>
                                <td>
                                    ${u.enabled
                ? '<span class="badge badge-success">Active</span>'
                : '<span class="badge badge-danger">Disabled</span>'
            }
                                </td>
                                <td>
                                    <div class="flex gap-2">
                                        <button class="btn btn-secondary btn-sm" onclick="showSubToken('${escapeHtml(u.sub_token)}', '${escapeHtml(subPath)}', '${escapeHtml(host)}')" title="Subscription URL">Sub URL</button>
                                        <button class="btn btn-secondary btn-sm" onclick="showUserLink(${u.id})" title="Sing-Box URL">Manual</button>
                                        <button class="btn btn-secondary btn-sm" onclick="resetUserHWID(${u.id})" title="Reset Device Limits">Reset HWIDs</button>
                                        <button class="btn btn-secondary btn-sm" onclick="toggleUserStatus(${u.id}, ${!u.enabled})">${u.enabled ? 'Disable' : 'Enable'}</button>
                                        <button class="btn btn-danger btn-sm" onclick="deleteUser(${u.id}, '${escapeHtml(u.username)}')">Delete</button>
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
        <div class="modal-overlay active" onclick="closeUserModal(event)">
            <div class="modal" onclick="event.stopPropagation()">
                <div class="modal-header">
                    <h3>Add New User</h3>
                    <button class="close-btn" onclick="closeUserModal()">×</button>
                </div>
                <div class="modal-body">
                    <form id="create-user-form">
                        <div class="form-group">
                            <label>Username</label>
                            <input type="text" id="new-username" placeholder="Leave empty to auto-generate">
                        </div>
                        <div class="form-group">
                            <label>Password</label>
                            <input type="text" id="new-password" placeholder="Leave empty to auto-generate">
                        </div>
                        <div class="grid grid-cols-2">
                            <div class="form-group">
                                <label>Traffic Limit (GB)</label>
                                <input type="number" id="new-traffic-limit" placeholder="0 = unlimited" value="0" min="0">
                            </div>
                            <div class="form-group">
                                <label>HWID Devices Limit</label>
                                <input type="number" id="new-hwid-limit" placeholder="0 = unlimited" value="0" min="0">
                            </div>
                        </div>
                        <div class="modal-footer">
                            <button type="button" class="btn btn-secondary" onclick="closeUserModal()">Cancel</button>
                            <button type="submit" class="btn btn-primary" id="create-user-btn">Create User</button>
                        </div>
                    </form>
                </div>
            </div>
        </div>
    `;

    document.getElementById('create-user-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        const btn = document.getElementById('create-user-btn');
        btn.disabled = true;

        const trafficGB = parseFloat(document.getElementById('new-traffic-limit').value) || 0;
        const hwidLimit = parseInt(document.getElementById('new-hwid-limit').value) || 0;

        try {
            await API.createUser({
                username: document.getElementById('new-username').value.trim(),
                password: document.getElementById('new-password').value,
                traffic_limit: Math.floor(trafficGB * 1024 * 1024 * 1024),
                hwid_limit: hwidLimit,
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
        Toast.success(`User ${enabled ? 'enabled' : 'disabled'} `);
        await loadUsers();
    } catch (error) {
        Toast.error(error.message);
    }
}

async function deleteUser(id, username) {
    if (!confirm(`Delete user "${username}" ? This cannot be undone.`)) return;

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
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6L6 18M6 6l12 12" /></svg>
                        </button>
                    </div>
                    <div class="qr-container">
                        ${link.qr_code ? `<img src="data:image/png;base64,${link.qr_code}" alt="QR Code">` : ''}
                        <div class="qr-uri">${escapeHtml(link.uri)}</div>
                        <button class="btn btn-primary btn-sm" onclick="copyToClipboard('${escapeHtml(link.uri)}')">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" /><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1" /></svg>
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
    if (!str) return '';
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
}

function showSubToken(token, subPath, host) {
    const url = `${host}/${subPath}/${token}`;
    const container = document.getElementById('user-modal-container');
    if (!container) return;

    container.innerHTML = `
        <div class="modal-overlay active" onclick="closeUserModal(event)">
            <div class="modal" onclick="event.stopPropagation()">
                <div class="modal-header">
                    <h3>Subscription URL</h3>
                    <button class="close-btn" onclick="closeUserModal()">×</button>
                </div>
                <div class="modal-body text-center">
                    <p class="mb-4 text-muted">Copy this strictly for use in the NaiveUI client.</p>
                    <code style="display:block;background:rgba(0,0,0,0.3);padding:1rem;border-radius:8px;word-break:break-all;margin-bottom:1rem;">
                        ${escapeHtml(url)}
                    </code>
                    <button class="btn btn-primary" onclick="copyToClipboard('${escapeHtml(url)}')">Copy to Clipboard</button>
                </div>
            </div>
        </div>
    `;
}

async function resetUserHWID(id) {
    if (!confirm('Are you sure you want to clear HWID lockouts for this user?')) return;
    try {
        await API.resetHWID(id);
        Toast.success('HWIDs cleared.');
    } catch (err) {
        Toast.error(err.message);
    }
}
