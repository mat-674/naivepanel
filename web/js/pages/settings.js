// ============================
// Settings Page
// ============================

function renderSettings() {
    return `
        <div class="page-header">
            <h2>Settings</h2>
            <p class="text-muted">Configure NaiveProxy server, panel, and subscription routing.</p>
        </div>
        <div class="glass-panel mt-4">
            <form id="settings-form">
                
                <h3 class="mb-4">Server Configuration</h3>
                <div class="grid grid-cols-2 mb-4">
                    <div class="form-group">
                        <label for="setting-domain">Domain Name</label>
                        <input type="text" id="setting-domain" placeholder="proxy.example.com">
                    </div>
                    <div class="form-group">
                        <label for="setting-port">Listen Port</label>
                        <input type="number" id="setting-port" placeholder="443" value="443" min="1" max="65535">
                    </div>
                </div>

                <h3 class="mt-4 mb-4">Routing & ACME</h3>
                <div class="grid grid-cols-2 mb-4">
                    <div class="form-group">
                        <label for="setting-subpath">Subscription Path</label>
                        <input type="text" id="setting-subpath" placeholder="sub">
                        <small class="text-muted">Users fetch subs at: <b>https://domain.com/&lt;path&gt;/token</b></small>
                        <div id="subpath-conflict-warning" class="text-warning hidden">⚠ Conflicts with published panel path</div>
                    </div>
                    <div class="form-group">
                        <label for="setting-tls-email">Let's Encrypt Email</label>
                        <input type="email" id="setting-tls-email" placeholder="admin@example.com">
                        <small class="text-muted">Optional. Leave blank for self-signed certs.</small>
                    </div>
                </div>
                
                <h3 class="mt-4 mb-4">Camouflage</h3>
                <div class="grid grid-cols-1 mb-4">
                    <div class="form-group">
                        <label for="setting-decoy">Decoy Site URL</label>
                        <input type="url" id="setting-decoy" placeholder="https://www.example.com">
                        <small class="text-muted">Requests that don't match NaiveProxy are forwarded here.</small>
                    </div>
                </div>

                <h3 class="mt-4 mb-4">Monitoring</h3>
                <div class="glass-panel" style="background: rgba(255,255,255,0.02); border: 1px solid rgba(255,255,255,0.05);">
                    <p class="text-muted" style="margin: 0; font-size: 0.9rem;">
                        <strong>Note:</strong> Traffic accounting is not yet implemented. The <code>traffic_limit</code>, <code>traffic_up</code>, and <code>traffic_down</code> fields are informational placeholders only and do not enforce any quota or track actual bandwidth usage.
                    </p>
                </div>

                <h3 class="mt-4 mb-4">Panel Maintenance</h3>
                <div class="glass-panel" style="background: rgba(255,255,255,0.02); border: 1px solid rgba(255,255,255,0.05);">
                    <div class="flex justify-between align-center">
                        <div>
                            <h4 style="margin:0; font-weight: 500;">Update NaivePanel</h4>
                            <p class="text-muted" style="margin: 0.25rem 0 0 0; font-size: 0.9rem;">Pull the latest version from GitHub and rebuild the panel. This will restart the service.</p>
                        </div>
                        <button type="button" class="btn btn-secondary" id="update-panel-btn" onclick="updatePanel()">Update Panel</button>
                    </div>
                </div>

                <div class="flex gap-4 mt-4 py-4 mt-6 border-t border-dark">
                    <button type="submit" class="btn btn-primary" id="save-settings-btn">Save & Apply Settings</button>
                    <button type="button" class="btn btn-secondary" onclick="initSettings()">Revert</button>
                </div>
            </form>
        </div>
    `;
}

async function updatePanel() {
    if (!confirm("Are you sure you want to update the panel? The service will restart and you may be disconnected briefly.")) return;

    const btn = document.getElementById('update-panel-btn');
    btn.disabled = true;

    // Show blocking modal
    showUpdateModal();

    try {
        // Start the update
        updateModalMessage('Starting update...', 'info');
        await API.serviceAction('update');

        // Update successful, start polling
        updateModalMessage('Waiting for panel to restart...', 'info');
        await pollPanelHealth();

    } catch (error) {
        // Initial update request failed
        updateModalMessage(`Failed to trigger update: ${error.message}`, 'error');
        btn.disabled = false;
    }
}

function showUpdateModal() {
    const container = document.getElementById('user-modal-container') || createModalContainer();

    container.innerHTML = `
        <div class="modal-overlay active" style="pointer-events: auto;">
            <div class="modal" onclick="event.stopPropagation()" style="pointer-events: auto;">
                <div class="modal-header">
                    <h3>Panel Update</h3>
                </div>
                <div class="modal-body" style="text-align: center; padding: 2rem;">
                    <div class="update-spinner"></div>
                    <p id="update-status-message" style="margin-top: 1.5rem; font-size: 1rem;">Preparing update...</p>
                </div>
                <div class="modal-footer" id="update-modal-footer" style="display: none;">
                    <button type="button" class="btn btn-secondary" onclick="closeUpdateModal()">Close</button>
                </div>
            </div>
        </div>
    `;
}

function createModalContainer() {
    const container = document.createElement('div');
    container.id = 'user-modal-container';
    document.body.appendChild(container);
    return container;
}

function updateModalMessage(message, type = 'info') {
    const msgEl = document.getElementById('update-status-message');
    if (!msgEl) return;

    msgEl.textContent = message;

    // Color coding based on type
    if (type === 'error') {
        msgEl.style.color = 'var(--danger)';
        showCloseButton();
    } else if (type === 'success') {
        msgEl.style.color = 'var(--success)';
    } else {
        msgEl.style.color = 'var(--text-main)';
    }
}

function showCloseButton() {
    const footer = document.getElementById('update-modal-footer');
    if (footer) footer.style.display = 'flex';
}

function closeUpdateModal() {
    const container = document.getElementById('user-modal-container');
    if (container) container.innerHTML = '';

    // Re-enable the update button
    const btn = document.getElementById('update-panel-btn');
    if (btn) btn.disabled = false;
}

async function pollPanelHealth() {
    const maxAttempts = 30; // 30 attempts × 2 seconds = 60 seconds
    const pollInterval = 2000; // 2 seconds

    for (let attempt = 1; attempt <= maxAttempts; attempt++) {
        await sleep(pollInterval);

        try {
            // Try to fetch panel status
            const response = await fetch('/api/status', {
                method: 'GET',
                headers: {
                    'Authorization': 'Bearer ' + localStorage.getItem('token')
                }
            });

            if (response.ok) {
                // Panel is back online
                updateModalMessage('Update complete! Redirecting...', 'success');
                Toast.success('Panel updated successfully!');

                // Brief delay before redirect
                await sleep(500);
                window.location.hash = '#/dashboard';
                closeUpdateModal();
                return;
            }
        } catch (error) {
            // Expected during restart - keep polling
            updateModalMessage(`Waiting for panel to restart... (${attempt}/${maxAttempts})`, 'info');
        }
    }

    // Timeout reached
    updateModalMessage(
        'Update may have succeeded, but the panel is not responding. Check the server logs or SSH in to verify.',
        'error'
    );
}

function sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
}

async function initSettings() {
    try {
        const res = await API.getSettings();
        const settings = res.data;

        document.getElementById('setting-domain').value = settings.domain || '';
        document.getElementById('setting-port').value = settings.port || 443;
        document.getElementById('setting-tls-email').value = settings.tls_email || '';
        document.getElementById('setting-decoy').value = settings.decoy_site || '';
        document.getElementById('setting-subpath').value = settings.sub_path || 'sub';

        // Attach input event listener for conflict warning
        const subpathInput = document.getElementById('setting-subpath');
        const warningEl = document.getElementById('subpath-conflict-warning');

        if (subpathInput && warningEl) {
            subpathInput.addEventListener('input', function() {
                // Normalize the input value the same way the submit handler does
                let inputValue = this.value.trim().replace(/^\/+|\/+$/g, '').toLowerCase();

                // Check if panel is published and paths conflict
                if (settings.panel_public === true) {
                    let panelPath = (settings.panel_public_path || '').trim().replace(/^\/+|\/+$/g, '').toLowerCase();

                    if (inputValue && panelPath && inputValue === panelPath) {
                        warningEl.classList.remove('hidden');
                    } else {
                        warningEl.classList.add('hidden');
                    }
                } else {
                    warningEl.classList.add('hidden');
                }
            });
        }

    } catch (error) {
        Toast.error('Failed to load settings: ' + error.message);
    }

    const form = document.getElementById('settings-form');
    if (form) {
        // Remove previous listener by cloning
        const newForm = form.cloneNode(true);
        form.parentNode.replaceChild(newForm, form);

        newForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const btn = document.getElementById('save-settings-btn');
            btn.disabled = true;

            try {
                let subPath = document.getElementById('setting-subpath').value.trim();
                subPath = subPath.replace(/^\/+|\/+$/g, ''); // Strip leading/trailing slashes

                await API.updateSettings({
                    domain: document.getElementById('setting-domain').value.trim(),
                    port: parseInt(document.getElementById('setting-port').value) || 443,
                    tls_email: document.getElementById('setting-tls-email').value.trim(),
                    decoy_site: document.getElementById('setting-decoy').value.trim(),
                    sub_path: subPath || 'sub',
                });

                Toast.success('Settings saved cleanly! Routing updated.');
            } catch (error) {
                Toast.error('Error: ' + error.message);
            } finally {
                btn.disabled = false;
            }
        });
    }
}
