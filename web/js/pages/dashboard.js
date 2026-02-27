// ============================
// Dashboard Page
// ============================

function renderDashboard() {
    return `
        <div class="page-header">
            <h2>Dashboard</h2>
            <p class="text-muted">Server overview and quick actions</p>
        </div>
        <div class="page-body mt-4">
            <div class="grid grid-cols-2" id="stats-grid">
                <div class="text-center text-muted">Loading...</div>
            </div>
            <div class="glass-panel mt-4" id="quick-actions-card" style="display:none;">
                <h3 class="mb-4">Quick Actions</h3>
                <div class="flex gap-4" id="quick-actions"></div>
            </div>
        </div>
    `;
}

async function initDashboard() {
    try {
        const res = await API.getStatus();
        const status = res.data;

        const statsGrid = document.getElementById('stats-grid');
        if (!statsGrid) return;

        statsGrid.innerHTML = `
            <div class="glass-panel stat-card">
                <div class="stat-title">Status</div>
                <div class="stat-value" style="color: ${status.running ? 'var(--success)' : 'var(--danger)'}">${status.running ? 'Running' : 'Stopped'}</div>
                <div class="text-muted" style="font-size:0.875rem;">${status.running ? 'PID: ' + status.pid : 'Service is not running'}</div>
            </div>

            <div class="glass-panel stat-card">
                <div class="stat-title">Active Users</div>
                <div class="stat-value text-primary">${status.user_count}</div>
                <div class="text-muted" style="font-size:0.875rem;">Proxy accounts</div>
            </div>

            <div class="glass-panel stat-card">
                <div class="stat-title">Uptime</div>
                <div class="stat-value">${status.uptime || '—'}</div>
                <div class="text-muted" style="font-size:0.875rem;">System: ${status.system_uptime || 'N/A'}</div>
            </div>

            <div class="glass-panel stat-card">
                <div class="stat-title">Global Traffic</div>
                <div class="stat-value text-warning">↑ ${formatBytes(status.total_up)}</div>
                <div class="text-muted" style="font-size:0.875rem;">↓ ${formatBytes(status.total_down)}</div>
            </div>

            <div class="glass-panel stat-card" style="grid-column: 1 / -1;">
                <div class="stat-title">System</div>
                <div class="stat-value" style="font-size:1.5rem;">${status.system_os} / ${status.system_arch}</div>
                <div class="text-muted" style="font-size:0.875rem;">NaivePanel v${status.version}</div>
            </div>
        `;

        // Quick actions
        const actionsCard = document.getElementById('quick-actions-card');
        const actionsDiv = document.getElementById('quick-actions');
        if (actionsCard && actionsDiv) {
            actionsCard.style.display = 'block';
            actionsDiv.innerHTML = `
                ${!status.running ? `
                    <button class="btn btn-primary" onclick="serviceAction('start')">Start Server</button>
                ` : `
                    <button class="btn btn-secondary" onclick="serviceAction('restart')">Restart Server</button>
                    <button class="btn btn-danger" onclick="serviceAction('stop')">Stop Server</button>
                `}
            `;
        }
    } catch (error) {
        const statsGrid = document.getElementById('stats-grid');
        if (statsGrid) {
            statsGrid.innerHTML = `
                <div class="glass-panel stat-card" style="grid-column: 1 / -1; border-color: var(--danger);">
                    <div class="stat-title text-danger">Connection Error</div>
                    <div class="stat-value" style="font-size:1rem;">${error.message}</div>
                    <div class="text-muted">Check that the NaivePanel backend is running</div>
                </div>
            `;
        }
    }
}

async function serviceAction(action) {
    try {
        await API.serviceAction(action);
        Toast.success(`Service ${action} successful`);
        setTimeout(() => initDashboard(), 1000);
    } catch (error) {
        Toast.error(error.message);
    }
}

function formatBytes(bytes) {
    if (!bytes || bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}
