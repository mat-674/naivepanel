// ============================
// Login Page
// ============================

function renderLogin() {
    return `
        <div class="setup-container">
            <div class="setup-card glass-panel" style="text-align:center;">
                <div class="nav-brand mb-4" style="font-size:2rem;">NaivePanel</div>
                <p class="text-muted mb-4">Sign in to manage your NaiveProxy nodes and users.</p>
                <form id="login-form">
                    <div class="form-group" style="text-align:left;">
                        <label for="username">Username</label>
                        <input type="text" id="username" placeholder="Enter username" autocomplete="username" autofocus>
                    </div>
                    <div class="form-group" style="text-align:left;">
                        <label for="password">Password</label>
                        <input type="password" id="password" placeholder="Enter password" autocomplete="current-password">
                    </div>
                    <button type="submit" class="btn btn-primary mt-4" style="width: 100%;" id="login-btn">
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
        btn.innerHTML = 'Signing in...';

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
            btn.innerHTML = 'Sign In';
        }
    });
}
