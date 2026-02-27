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
    const originalText = btn.textContent;
    btn.disabled = true;
    btn.textContent = 'Updating...';

    try {
        await API.serviceAction('update');
        Toast.success('Update started! The panel will restart shortly.', 5000);

        // Wait a bit, then try to reload to see if it's back up
        setTimeout(() => {
            window.location.reload();
        }, 5000);
    } catch (error) {
        Toast.error('Failed to trigger update: ' + error.message);
        btn.disabled = false;
        btn.textContent = originalText;
    }
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
