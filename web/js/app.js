// ============================
// App Router & Layout
// ============================

const ICONS = {
    dashboard: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/></svg>',
    users: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17 21v-2a4 4 0 00-4-4H5a4 4 0 00-4-4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 00-3-3.87"/><path d="M16 3.13a4 4 0 010 7.75"/></svg>',
    settings: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 010 2.83 2 2 0 01-2.83 0l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 01-2 2 2 2 0 01-2-2v-.09A1.65 1.65 0 009 19.4a1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 01-2.83 0 2 2 0 010-2.83l.06-.06A1.65 1.65 0 004.68 15a1.65 1.65 0 00-1.51-1H3a2 2 0 01-2-2 2 2 0 012-2h.09A1.65 1.65 0 004.6 9a1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 010-2.83 2 2 0 012.83 0l.06.06A1.65 1.65 0 009 4.68a1.65 1.65 0 001-1.51V3a2 2 0 012-2 2 2 0 012 2v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 012.83 0 2 2 0 010 2.83l-.06.06A1.65 1.65 0 0019.32 9a1.65 1.65 0 001.51 1H21a2 2 0 012 2 2 2 0 01-2 2h-.09a1.65 1.65 0 00-1.51 1z"/></svg>',
    logout: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M9 21H5a2 2 0 01-2-2V5a2 2 0 012-2h4"/><polyline points="16 17 21 12 16 7"/><line x1="21" y1="12" x2="9" y2="12"/></svg>',
};

const routes = {
    '/login': { render: renderLogin, init: initLogin, auth: false },
    '/dashboard': { render: renderDashboard, init: initDashboard, auth: true },
    '/users': { render: renderUsers, init: initUsers, auth: true },
    '/settings': { render: renderSettings, init: initSettings, auth: true },
};

function renderLayout(pageContent, activePage) {
    return `
        <div class="layout">
            <nav class="sidebar">
                <div class="sidebar-header">
                    <h1>NaivePanel</h1>
                    <span>NaiveProxy Management</span>
                </div>
                <div class="sidebar-nav">
                    <a class="nav-item ${activePage === '/dashboard' ? 'active' : ''}" href="#/dashboard">
                        ${ICONS.dashboard}
                        <span>Dashboard</span>
                    </a>
                    <a class="nav-item ${activePage === '/users' ? 'active' : ''}" href="#/users">
                        ${ICONS.users}
                        <span>Users</span>
                    </a>
                    <a class="nav-item ${activePage === '/settings' ? 'active' : ''}" href="#/settings">
                        ${ICONS.settings}
                        <span>Settings</span>
                    </a>
                </div>
                <div class="sidebar-footer">
                    <a class="nav-item logout" href="#" onclick="API.logout(); return false;">
                        ${ICONS.logout}
                        <span>Logout</span>
                    </a>
                </div>
            </nav>
            <main class="main-content">
                ${pageContent}
            </main>
        </div>
    `;
}

function navigateTo(path) {
    const route = routes[path];
    if (!route) {
        window.location.hash = '#/dashboard';
        return;
    }

    // Auth check
    if (route.auth && !API.isAuthenticated()) {
        window.location.hash = '#/login';
        return;
    }

    if (!route.auth && path === '/login' && API.isAuthenticated()) {
        window.location.hash = '#/dashboard';
        return;
    }

    const app = document.getElementById('app');
    if (!app) return;

    const content = route.render();

    if (route.auth) {
        app.innerHTML = renderLayout(content, path);
    } else {
        app.innerHTML = content;
    }

    // Init page after render
    if (route.init) {
        setTimeout(() => route.init(), 0);
    }
}

function handleRoute() {
    let hash = window.location.hash.replace('#', '') || '/dashboard';
    if (hash === '/') hash = '/dashboard';
    navigateTo(hash);
}

// Listen for hash changes
window.addEventListener('hashchange', handleRoute);

// Initial route
window.addEventListener('DOMContentLoaded', () => {
    if (!window.location.hash || window.location.hash === '#/' || window.location.hash === '#') {
        window.location.hash = API.isAuthenticated() ? '#/dashboard' : '#/login';
    } else {
        handleRoute();
    }
});
