// ============================
// Users Management Page
// ============================

// Global reference for the current modal Escape handler to prevent listener leaks
let currentModalEscHandler = null;

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

        let subscriptionBaseURL = "";
        try {
            const setRes = await API.getSettings();
            if (setRes.data) subscriptionBaseURL = getSubscriptionBaseURL(setRes.data);
        } catch (e) { }

        tableDiv.innerHTML = `
            <div class="table-container">
                <table>
                    <thead>
                        <tr>
                            <th>Username</th>
                            <th>Traffic Limit</th>
                            <th>HWID Limit</th>
                            <th>Expires</th>
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
                                <td>${u.expires_at ? new Date(u.expires_at).toLocaleString() : '<span class="text-muted">Never</span>'}</td>
                                <td>
                                    ${u.enabled
                ? '<span class="badge badge-success">Active</span>'
                : '<span class="badge badge-danger">Disabled</span>'
            }
                                </td>
                                <td>
                                    <div class="flex gap-2">
                                        <button class="btn btn-secondary btn-sm" onclick="showSubToken('${jsAttr(u.sub_token)}', '${jsAttr(subscriptionBaseURL)}')" title="Subscription URL">Sub URL</button>
                                        <button class="btn btn-secondary btn-sm" onclick="showUserLink(${u.id})" title="Sing-Box URL">Manual</button>
                                        <button class="btn btn-secondary btn-sm" onclick="showEditUserModal(${escapeHtml(JSON.stringify(u))})" title="Edit User">Edit</button>
                                        <button class="btn btn-secondary btn-sm" onclick="resetUserHWID(${u.id})" title="Reset Device Limits">Reset HWIDs</button>
                                        <button class="btn btn-secondary btn-sm" onclick="toggleUserStatus(${u.id}, ${!u.enabled})">${u.enabled ? 'Disable' : 'Enable'}</button>
                                        <button class="btn btn-danger btn-sm" onclick="deleteUser(${u.id}, '${jsAttr(u.username)}')">Delete</button>
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
                            <input type="text" id="new-username" placeholder="Leave empty to auto-generate" maxlength="64" pattern="[A-Za-z0-9._~-]*" title="Letters, digits and - . _ ~ only">
                        </div>
                        <div class="form-group">
                            <label>Password</label>
                            <input type="text" id="new-password" placeholder="Leave empty to auto-generate" maxlength="128" pattern="[A-Za-z0-9._~-]*" title="Letters, digits and - . _ ~ only">
                            <small class="text-muted">Allowed characters: letters, digits, and - . _ ~</small>
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
                        <div class="form-group mb-4 mt-2">
                            <label>Expiration Date</label>
                            <input type="datetime-local" id="new-expires-at">
                            <small class="text-muted">Leave blank for no expiration.</small>
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

    // Attach Escape key handler
    currentModalEscHandler = function(e) {
        if (e.key === 'Escape') {
            closeUserModal();
        }
    };
    document.addEventListener('keydown', currentModalEscHandler);

    document.getElementById('create-user-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        const btn = document.getElementById('create-user-btn');
        btn.disabled = true;

        const trafficGB = parseFloat(document.getElementById('new-traffic-limit').value) || 0;
        const hwidLimit = parseInt(document.getElementById('new-hwid-limit').value) || 0;
        const expiresAtStr = document.getElementById('new-expires-at').value;

        let payload = {
            username: document.getElementById('new-username').value.trim(),
            password: document.getElementById('new-password').value,
            traffic_limit: Math.floor(trafficGB * 1024 * 1024 * 1024),
            hwid_limit: hwidLimit,
        };

        if (expiresAtStr) {
            payload.expires_at = Math.floor(new Date(expiresAtStr).getTime() / 1000);
        } else {
            // Send null or omitted to indicate no expiration
            payload.expires_at = null;
        }

        try {
            const response = await API.createUser(payload);
            Toast.success('User created! Fetching connection link...');
            closeUserModal();
            if (response.data && response.data.id) {
                await showCredentialsModal(response.data);
            }
            await loadUsers();
        } catch (error) {
            Toast.error(error.message);
            btn.disabled = false;
        }
    });
}

function showEditUserModal(user) {
    const container = document.getElementById('user-modal-container');
    if (!container) return;

    const trafficGB = user.traffic_limit ? (user.traffic_limit / (1024 * 1024 * 1024)).toFixed(2) : 0;

    // Format date for datetime-local "YYYY-MM-DDThh:mm"
    let tzoffset = (new Date()).getTimezoneOffset() * 60000; // offset in milliseconds
    let localISOTime = '';
    if (user.expires_at) {
        localISOTime = (new Date(new Date(user.expires_at) - tzoffset)).toISOString().slice(0, 16);
    }

    container.innerHTML = `
        <div class="modal-overlay active" onclick="closeUserModal(event)">
            <div class="modal" onclick="event.stopPropagation()">
                <div class="modal-header">
                    <h3>Edit User: ${escapeHtml(user.username)}</h3>
                    <button class="close-btn" onclick="closeUserModal()">×</button>
                </div>
                <div class="modal-body">
                    <form id="edit-user-form">
                        <div class="form-group">
                            <label>Username</label>
                            <input type="text" value="${escapeHtml(user.username)}" disabled class="bg-dark text-muted cursor-not-allowed">
                            <small class="text-muted">Username cannot be changed as it is used for proxy authentication.</small>
                        </div>
                        <div class="form-group">
                            <label>New Password</label>
                            <input type="text" id="edit-password" placeholder="Leave blank to keep current password" maxlength="128" pattern="[A-Za-z0-9._~-]*" title="Letters, digits and - . _ ~ only">
                            <small class="text-muted">Allowed characters: letters, digits, and - . _ ~</small>
                        </div>
                        <div class="grid grid-cols-2">
                            <div class="form-group">
                                <label>Traffic Limit (GB)</label>
                                <input type="number" id="edit-traffic-limit" placeholder="0 = unlimited" value="${trafficGB}" min="0" step="0.01">
                            </div>
                            <div class="form-group">
                                <label>HWID Devices Limit</label>
                                <input type="number" id="edit-hwid-limit" placeholder="0 = unlimited" value="${user.hwid_limit}" min="0">
                            </div>
                        </div>
                        <div class="form-group mb-4 mt-2">
                            <label>Expiration Date</label>
                            <input type="datetime-local" id="edit-expires-at" value="${localISOTime}">
                            <small class="text-muted">Clear to remove expiration.</small>
                        </div>
                        <div class="modal-footer">
                            <button type="button" class="btn btn-secondary" onclick="closeUserModal()">Cancel</button>
                            <button type="submit" class="btn btn-primary" id="edit-user-btn">Save Changes</button>
                        </div>
                    </form>
                </div>
            </div>
        </div>
    `;

    // Attach Escape key handler
    currentModalEscHandler = function(e) {
        if (e.key === 'Escape') {
            closeUserModal();
        }
    };
    document.addEventListener('keydown', currentModalEscHandler);

    document.getElementById('edit-user-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        const btn = document.getElementById('edit-user-btn');
        btn.disabled = true;

        const tf = parseFloat(document.getElementById('edit-traffic-limit').value) || 0;
        const hwidLimit = parseInt(document.getElementById('edit-hwid-limit').value) || 0;
        const newPassword = document.getElementById('edit-password').value;
        const expiresAtStr = document.getElementById('edit-expires-at').value;

        const updateData = {
            traffic_limit: Math.floor(tf * 1024 * 1024 * 1024),
            hwid_limit: hwidLimit,
        };

        if (expiresAtStr) {
            updateData.expires_at = Math.floor(new Date(expiresAtStr).getTime() / 1000);
        } else {
            // The backend treats explicit null as "clear expiry" and an omitted
            // key as "leave unchanged" (OptionalInt64 tri-state), so send null here.
            updateData.expires_at = null;
        }

        if (newPassword.trim() !== '') {
            updateData.password = newPassword;
        }

        try {
            await API.updateUser(user.id, updateData);
            Toast.success('User updated successfully');
            closeUserModal();
            await loadUsers();
        } catch (error) {
            Toast.error(error.message);
            btn.disabled = false;
        }
    });
}

async function showCredentialsModal(user) {
    try {
        // Fetch the connection URI and QR code
        const res = await API.getUserLink(user.id);
        const link = res.data;

        const container = document.getElementById('user-modal-container');
        if (!container) return;

        container.innerHTML = `
            <div class="modal-overlay active" onclick="event.stopPropagation()">
                <div class="modal" onclick="event.stopPropagation()" style="border: 2px solid #10b981; max-width: 600px;">
                    <div class="modal-header" style="background: linear-gradient(135deg, rgba(16, 185, 129, 0.1) 0%, rgba(5, 150, 105, 0.05) 100%);">
                        <div style="display: flex; align-items: center; gap: 0.75rem;">
                            <svg viewBox="0 0 24 24" fill="none" stroke="#10b981" stroke-width="2" style="width: 28px; height: 28px;">
                                <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path>
                                <polyline points="22 4 12 14.01 9 11.01"></polyline>
                            </svg>
                            <h3 style="margin: 0;">User Created Successfully!</h3>
                        </div>
                        <button class="close-btn" onclick="closeCredentialsModal()">×</button>
                    </div>
                    <div class="modal-body">
                        <p style="margin-bottom: 1.5rem; color: #f59e0b; font-weight: 500;">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width: 18px; height: 18px; display: inline; vertical-align: text-bottom; margin-right: 0.25rem;">
                                <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"></path>
                                <line x1="12" y1="9" x2="12" y2="13"></line>
                                <line x1="12" y1="17" x2="12.01" y2="17"></line>
                            </svg>
                            Save these credentials now! They won't be shown again.
                        </p>

                        <div style="background: rgba(0,0,0,0.3); padding: 1.25rem; border-radius: 8px; margin-bottom: 1rem;">
                            <div style="margin-bottom: 1rem;">
                                <label style="display: block; font-weight: 600; margin-bottom: 0.5rem; color: #9ca3af;">Username</label>
                                <code style="display: block; background: rgba(0,0,0,0.4); padding: 0.75rem; border-radius: 6px; font-size: 0.95rem;">${escapeHtml(user.username)}</code>
                            </div>

                            <div style="margin-bottom: 1rem;">
                                <label style="display: block; font-weight: 600; margin-bottom: 0.5rem; color: #9ca3af;">Password</label>
                                <div style="display: flex; gap: 0.5rem; align-items: center;">
                                    <input type="password" id="credential-password" value="${escapeHtml(user.password)}" readonly style="flex: 1; background: rgba(0,0,0,0.4); padding: 0.75rem; border-radius: 6px; border: none; color: inherit; font-family: monospace; font-size: 0.95rem;">
                                    <button class="btn btn-secondary btn-sm" onclick="togglePasswordVisibility()" id="toggle-password-btn" title="Show/Hide Password">
                                        <svg id="eye-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width: 18px; height: 18px;">
                                            <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path>
                                            <circle cx="12" cy="12" r="3"></circle>
                                        </svg>
                                    </button>
                                </div>
                            </div>

                            <div style="margin-bottom: 0;">
                                <label style="display: block; font-weight: 600; margin-bottom: 0.5rem; color: #9ca3af;">Connection URI</label>
                                <code style="display: block; background: rgba(0,0,0,0.4); padding: 0.75rem; border-radius: 6px; font-size: 0.85rem; word-break: break-all;">${escapeHtml(link.uri)}</code>
                            </div>
                        </div>

                        ${link.qr_code ? `
                            <div style="text-align: center; margin-bottom: 1.5rem;">
                                <img src="data:image/png;base64,${link.qr_code}" alt="QR Code" style="max-width: 200px; border-radius: 8px;">
                            </div>
                        ` : ''}

                        <div style="display: flex; gap: 0.75rem; margin-bottom: 1.5rem;">
                            <button class="btn btn-secondary" onclick="copyCredentialsToClipboard('${jsAttr(user.username)}', '${jsAttr(user.password)}', '${jsAttr(link.uri)}')" style="flex: 1;">
                                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width: 16px; height: 16px; margin-right: 0.5rem; display: inline; vertical-align: text-bottom;">
                                    <rect x="9" y="9" width="13" height="13" rx="2" />
                                    <path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1" />
                                </svg>
                                Copy All Credentials
                            </button>
                            <button class="btn btn-secondary" onclick="copyToClipboard('${jsAttr(link.uri)}')" style="flex: 1;">
                                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width: 16px; height: 16px; margin-right: 0.5rem; display: inline; vertical-align: text-bottom;">
                                    <rect x="9" y="9" width="13" height="13" rx="2" />
                                    <path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1" />
                                </svg>
                                Copy URI Only
                            </button>
                        </div>

                        <div style="text-align: center;">
                            <button class="btn btn-primary" onclick="closeCredentialsModal()" style="min-width: 200px; padding: 0.75rem 1.5rem;">
                                I've Saved These Credentials
                            </button>
                        </div>
                    </div>
                </div>
            </div>
        `;

        // Attach Escape key handler
        currentModalEscHandler = function(e) {
            if (e.key === 'Escape') {
                closeCredentialsModal();
            }
        };
        document.addEventListener('keydown', currentModalEscHandler);
    } catch (error) {
        Toast.error('Failed to fetch connection details: ' + error.message);
    }
}

function togglePasswordVisibility() {
    const passwordInput = document.getElementById('credential-password');
    const eyeIcon = document.getElementById('eye-icon');

    if (passwordInput.type === 'password') {
        passwordInput.type = 'text';
        eyeIcon.innerHTML = `
            <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"></path>
            <line x1="1" y1="1" x2="23" y2="23"></line>
        `;
    } else {
        passwordInput.type = 'password';
        eyeIcon.innerHTML = `
            <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path>
            <circle cx="12" cy="12" r="3"></circle>
        `;
    }
}

function copyCredentialsToClipboard(username, password, uri) {
    const text = `Username: ${username}\nPassword: ${password}\nURI: ${uri}`;
    navigator.clipboard.writeText(text).then(() => {
        Toast.success('All credentials copied to clipboard!');
    }).catch(() => {
        // Fallback
        const textarea = document.createElement('textarea');
        textarea.value = text;
        document.body.appendChild(textarea);
        textarea.select();
        document.execCommand('copy');
        document.body.removeChild(textarea);
        Toast.success('All credentials copied to clipboard!');
    });
}

function closeCredentialsModal() {
    const container = document.getElementById('user-modal-container');
    if (container) {
        container.innerHTML = '';
        if (currentModalEscHandler) {
            document.removeEventListener('keydown', currentModalEscHandler);
            currentModalEscHandler = null;
        }
    }
}

function closeUserModal(event) {
    if (event && event.target !== event.currentTarget) return;
    const container = document.getElementById('user-modal-container');
    if (container) {
        container.innerHTML = '';
        // Remove the Escape key handler to prevent listener leaks
        if (currentModalEscHandler) {
            document.removeEventListener('keydown', currentModalEscHandler);
            currentModalEscHandler = null;
        }
    }
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
                        <button class="btn btn-primary btn-sm" onclick="copyToClipboard('${jsAttr(link.uri)}')">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" /><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1" /></svg>
                            Copy Link
                        </button>
                    </div>
                </div>
            </div>
        `;

        // Attach Escape key handler
        currentModalEscHandler = function(e) {
            if (e.key === 'Escape') {
                closeUserModal();
            }
        };
        document.addEventListener('keydown', currentModalEscHandler);
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

// escapeHtml makes a string safe to interpolate into HTML text or a
// double-quoted attribute value. It escapes all five significant characters,
// including both quote styles (the DOM textContent trick left quotes intact,
// which allowed breakout from value="..." attributes).
function escapeHtml(str) {
    if (str === null || str === undefined) return '';
    return String(str)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
}

// jsAttr makes a string safe to embed as a single-quoted JS string literal
// inside a double-quoted inline handler, e.g. onclick="fn('${jsAttr(x)}')".
// The browser HTML-decodes the attribute before the JS engine parses it, so
// HTML-escaping alone would not stop a quote from breaking out of the JS
// string. The value is first escaped for the JS string context and then
// HTML-encoded so it survives the surrounding double-quoted attribute.
function jsAttr(str) {
    if (str === null || str === undefined) return '';
    const jsEscaped = String(str)
        .replace(/\\/g, '\\\\')
        .replace(/'/g, "\\'")
        .replace(/\r/g, '\\r')
        .replace(/\n/g, '\\n')
        .replace(/2028/g, '\u2028')
        .replace(/2029/g, '\u2029');
    return jsEscaped
        .replace(/&/g, '&amp;')
        .replace(/"/g, '&quot;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;');
}

function getSubscriptionBaseURL(settings) {
    const domain = (settings.domain || '').trim();
    const subPath = (settings.sub_path || 'sub').replace(/^\/+|\/+$/g, '');
    const port = Number(settings.port) || 443;

    if (!domain || !subPath) return '';

    const authority = port === 443 ? domain : `${domain}:${port}`;
    return `https://${authority}/${subPath}`;
}

function showSubToken(token, subscriptionBaseURL) {
    if (!subscriptionBaseURL) {
        Toast.error('Configure a public domain before creating subscription URLs.');
        return;
    }

    const url = `${subscriptionBaseURL}/${encodeURIComponent(token)}`;
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
                    <button class="btn btn-primary" onclick="copyToClipboard('${jsAttr(url)}')">Copy to Clipboard</button>
                </div>
            </div>
        </div>
    `;

    // Attach Escape key handler
    currentModalEscHandler = function(e) {
        if (e.key === 'Escape') {
            closeUserModal();
        }
    };
    document.addEventListener('keydown', currentModalEscHandler);
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
