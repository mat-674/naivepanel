// ============================
// App Router & Layout
// ============================

const routes = {
    '/login': { render: renderLogin, init: initLogin, auth: false },
    '/dashboard': { render: renderDashboard, init: initDashboard, auth: true },
    '/users': { render: renderUsers, init: initUsers, auth: true },
    '/settings': { render: renderSettings, init: initSettings, auth: true },
};

function updateNavigation(activePath) {
    const nav = document.getElementById('main-nav');
    if (!nav) return;

    // Hide nav on login page, show on others
    if (activePath === '/login') {
        nav.style.display = 'none';
        return;
    }

    nav.style.display = 'flex';

    // Update active state
    document.querySelectorAll('.nav-link').forEach(link => {
        link.classList.remove('active');
    });

    let navId = 'nav-dashboard';
    if (activePath.startsWith('/users')) navId = 'nav-users';
    if (activePath.startsWith('/settings')) navId = 'nav-settings';

    const activeLink = document.getElementById(navId);
    if (activeLink) activeLink.classList.add('active');
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

    // Update nav visibility and active state
    updateNavigation(path);

    // Render content
    app.innerHTML = `<div class="fade-enter-active">${route.render()}</div>`;

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
