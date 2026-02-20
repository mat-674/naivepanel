// ============================
// Login Page
// ============================

function renderLogin() {
    return `
        <div class="login-wrapper">
            <div class="login-card">
                <div class="login-logo">
                    <h1>NaivePanel</h1>
                    <p>NaiveProxy Management Panel</p>
                </div>
                <form id="login-form">
                    <div class="form-group">
                        <label class="form-label" for="username">Username</label>
                        <input type="text" id="username" class="form-input" placeholder="Enter username" autocomplete="username" autofocus>
                    </div>
                    <div class="form-group">
                        <label class="form-label" for="password">Password</label>
                        <input type="password" id="password" class="form-input" placeholder="Enter password" autocomplete="current-password">
                    </div>
                    <button type="submit" class="btn btn-primary" style="width: 100%; margin-top: 8px;" id="login-btn">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M15 3h4a2 2 0 012 2v14a2 2 0 01-2 2h-4M10 17l5-5-5-5M15 12H3"/></svg>
                        Sign In
                    </button>
                </form>
            </div>
        </div>
    `;
}

function initLogin() {
    const form = document.getElementById('login-form');
    if (!form) return;

    form.addEventListener('submit', async (e) => {
        e.preventDefault();
        const btn = document.getElementById('login-btn');
        const username = document.getElementById('username').value.trim();
        const password = document.getElementById('password').value;

        if (!username || !password) {
            Toast.error('Please enter username and password');
            return;
        }

        btn.disabled = true;
        btn.innerHTML = '<span class="spinner" style="width:16px;height:16px;border-width:2px;"></span> Signing in...';

        try {
            await API.login(username, password);
            Toast.success('Login successful!');
            setTimeout(() => {
                window.location.hash = '#/dashboard';
            }, 300);
        } catch (error) {
            Toast.error(error.message || 'Login failed');
        } finally {
            btn.disabled = false;
            btn.innerHTML = `
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M15 3h4a2 2 0 012 2v14a2 2 0 01-2 2h-4M10 17l5-5-5-5M15 12H3"/></svg>
                Sign In
            `;
        }
    });
}
