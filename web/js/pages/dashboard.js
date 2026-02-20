// ============================
// Dashboard Page
// ============================

function renderDashboard() {
    return `
        <div class="page-header">
            <h2>Dashboard</h2>
            <p>Server overview and quick actions</p>
        </div>
        <div class="page-body">
            <div class="stats-grid" id="stats-grid">
                <div class="loading"><div class="spinner"></div></div>
            </div>
            <div class="card" id="quick-actions-card" style="display:none;">
                <div class="card-title">Quick Actions</div>
                <div class="quick-actions" id="quick-actions"></div>
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
            <div class="card stat-card ${status.running ? 'success' : 'danger'}">
                <div class="stat-icon ${status.running ? 'success' : 'danger'}">
                    ${status.running
                ? '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 11.08V12a10 10 0 11-5.93-9.14"/><path d="M22 4L12 14.01l-3-3"/></svg>'
                : '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M15 9l-6 6M9 9l6 6"/></svg>'
            }
                </div>
                <div class="card-title">Status</div>
                <div class="card-value">${status.running ? 'Running' : 'Stopped'}</div>
                <div class="card-footer">${status.running ? 'PID: ' + status.pid : 'Service is not running'}</div>
            </div>

            <div class="card stat-card accent">
                <div class="stat-icon accent">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17 21v-2a4 4 0 00-4-4H5a4 4 0 00-4-4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 00-3-3.87"/><path d="M16 3.13a4 4 0 010 7.75"/></svg>
                </div>
                <div class="card-title">Users</div>
                <div class="card-value">${status.user_count}</div>
                <div class="card-footer">Active proxy accounts</div>
            </div>

            <div class="card stat-card info">
                <div class="stat-icon info">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
                </div>
                <div class="card-title">Uptime</div>
                <div class="card-value">${status.uptime || '—'}</div>
                <div class="card-footer">System: ${status.system_uptime || 'N/A'}</div>
            </div>

            <div class="card stat-card warning">
                <div class="stat-icon warning">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg>
                </div>
                <div class="card-title">Traffic</div>
                <div class="card-value">↑ ${formatBytes(status.total_up)}</div>
                <div class="card-footer">↓ ${formatBytes(status.total_down)}</div>
            </div>

            <div class="card stat-card info">
                <div class="stat-icon info">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="3" width="20" height="14" rx="2"/><path d="M8 21h8M12 17v4"/></svg>
                </div>
                <div class="card-title">System</div>
                <div class="card-value" style="font-size:18px;">${status.system_os} / ${status.system_arch}</div>
                <div class="card-footer">NaivePanel v${status.version}</div>
            </div>
        `;

        // Quick actions
        const actionsCard = document.getElementById('quick-actions-card');
        const actionsDiv = document.getElementById('quick-actions');
        if (actionsCard && actionsDiv) {
            actionsCard.style.display = 'block';
            actionsDiv.innerHTML = `
                ${!status.running ? `
                    <button class="btn btn-primary btn-sm" onclick="serviceAction('start')">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="5 3 19 12 5 21 5 3"/></svg>
                        Start
                    </button>
                ` : `
                    <button class="btn btn-outline btn-sm" onclick="serviceAction('restart')">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M1 4v6h6M23 20v-6h-6"/><path d="M20.49 9A9 9 0 005.64 5.64L1 10m22 4l-4.64 4.36A9 9 0 013.51 15"/></svg>
                        Restart
                    </button>
                    <button class="btn btn-danger btn-sm" onclick="serviceAction('stop')">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="6" y="6" width="12" height="12"/></svg>
                        Stop
                    </button>
                `}
            `;
        }
    } catch (error) {
        const statsGrid = document.getElementById('stats-grid');
        if (statsGrid) {
            statsGrid.innerHTML = `
                <div class="card stat-card danger" style="grid-column: 1 / -1;">
                    <div class="card-title">Connection Error</div>
                    <div class="card-value" style="font-size:16px;">${error.message}</div>
                    <div class="card-footer">Check that the NaivePanel backend is running</div>
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
