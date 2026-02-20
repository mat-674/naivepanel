// ============================
// Settings Page
// ============================

function renderSettings() {
    return `
        <div class="page-header">
            <h2>Settings</h2>
            <p>Configure NaiveProxy server and panel</p>
        </div>
        <div class="page-body">
            <form id="settings-form">
                <div class="settings-section">
                    <h3>Server Configuration</h3>
                    <div class="settings-grid">
                        <div class="form-group">
                            <label class="form-label" for="setting-domain">Domain</label>
                            <input type="text" class="form-input" id="setting-domain" placeholder="example.com">
                            <div class="form-hint">Your server's domain name</div>
                        </div>
                        <div class="form-group">
                            <label class="form-label" for="setting-port">Listen Port</label>
                            <input type="number" class="form-input" id="setting-port" placeholder="443" value="443" min="1" max="65535">
                            <div class="form-hint">Port NaiveProxy listens on (default: 443)</div>
                        </div>
                    </div>
                </div>

                <div class="settings-section">
                    <h3>TLS / ACME (Let's Encrypt)</h3>
                    <div class="settings-grid">
                        <div class="form-group full">
                            <label class="form-label" for="setting-tls-email">ACME Email</label>
                            <input type="email" class="form-input" id="setting-tls-email" placeholder="admin@example.com">
                            <div class="form-hint">Email for Let's Encrypt certificate registration. Leave empty for self-signed.</div>
                        </div>
                    </div>
                </div>

                <div class="settings-section">
                    <h3>Decoy Site (Camouflage)</h3>
                    <div class="settings-grid">
                        <div class="form-group full">
                            <label class="form-label" for="setting-decoy">Decoy Site URL</label>
                            <input type="text" class="form-input" id="setting-decoy" placeholder="https://www.example.com">
                            <div class="form-hint">Website to display when probed. Requests that don't look like proxy traffic will be forwarded here.</div>
                        </div>
                    </div>
                </div>

                <div class="btn-group">
                    <button type="submit" class="btn btn-primary" id="save-settings-btn">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M19 21H5a2 2 0 01-2-2V5a2 2 0 012-2h11l5 5v11a2 2 0 01-2 2z"/><polyline points="17 21 17 13 7 13 7 21"/><polyline points="7 3 7 8 15 8"/></svg>
                        Save & Apply
                    </button>
                    <button type="button" class="btn btn-outline" onclick="initSettings()">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M1 4v6h6M23 20v-6h-6"/><path d="M20.49 9A9 9 0 005.64 5.64L1 10m22 4l-4.64 4.36A9 9 0 013.51 15"/></svg>
                        Reset
                    </button>
                </div>
            </form>
        </div>
    `;
}

async function initSettings() {
    try {
        const res = await API.getSettings();
        const settings = res.data;

        const domainInput = document.getElementById('setting-domain');
        const portInput = document.getElementById('setting-port');
        const emailInput = document.getElementById('setting-tls-email');
        const decoyInput = document.getElementById('setting-decoy');

        if (domainInput) domainInput.value = settings.domain || '';
        if (portInput) portInput.value = settings.port || 443;
        if (emailInput) emailInput.value = settings.tls_email || '';
        if (decoyInput) decoyInput.value = settings.decoy_site || '';

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
                await API.updateSettings({
                    domain: document.getElementById('setting-domain').value.trim(),
                    port: parseInt(document.getElementById('setting-port').value) || 443,
                    tls_email: document.getElementById('setting-tls-email').value.trim(),
                    decoy_site: document.getElementById('setting-decoy').value.trim(),
                });

                Toast.success('Settings saved! Caddyfile regenerated.');
            } catch (error) {
                Toast.error(error.message);
            } finally {
                btn.disabled = false;
            }
        });
    }
}
