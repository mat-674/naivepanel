// ============================
// API Client
// ============================

const API = {
    baseURL: '',

    getToken() {
        return localStorage.getItem('naivepanel_token');
    },

    setToken(token) {
        localStorage.setItem('naivepanel_token', token);
    },

    clearToken() {
        localStorage.removeItem('naivepanel_token');
    },

    isAuthenticated() {
        return !!this.getToken();
    },

    async request(endpoint, options = {}) {
        const url = `${this.baseURL}${endpoint}`;
        const headers = {
            'Content-Type': 'application/json',
            ...options.headers,
        };

        const token = this.getToken();
        if (token) {
            headers['Authorization'] = `Bearer ${token}`;
        }

        try {
            const response = await fetch(url, {
                ...options,
                headers,
            });

            if (response.status === 401) {
                this.clearToken();
                window.location.hash = '#/login';
                throw new Error('Session expired. Please login again.');
            }

            const data = await response.json();

            if (!response.ok) {
                throw new Error(data.message || `HTTP ${response.status}`);
            }

            return data;
        } catch (error) {
            if (error.message === 'Session expired. Please login again.') {
                throw error;
            }
            throw error;
        }
    },

    // Auth
    async login(username, password) {
        const data = await this.request('/api/login', {
            method: 'POST',
            body: JSON.stringify({ username, password }),
        });
        if (data.success && data.data && data.data.token) {
            this.setToken(data.data.token);
        }
        return data;
    },

    logout() {
        this.clearToken();
        window.location.hash = '#/login';
    },

    // Users
    async getUsers() {
        return this.request('/api/users');
    },

    async createUser(userData) {
        return this.request('/api/users', {
            method: 'POST',
            body: JSON.stringify(userData),
        });
    },

    async updateUser(id, userData) {
        return this.request(`/api/users/${id}`, {
            method: 'PUT',
            body: JSON.stringify(userData),
        });
    },

    async deleteUser(id) {
        return this.request(`/api/users/${id}`, {
            method: 'DELETE',
        });
    },

    async getUserLink(id) {
        return this.request(`/api/users/${id}/link`);
    },

    // Settings
    async getSettings() {
        return this.request('/api/settings');
    },

    async updateSettings(settings) {
        return this.request('/api/settings', {
            method: 'PUT',
            body: JSON.stringify(settings),
        });
    },

    // Status
    async getStatus() {
        return this.request('/api/status');
    },

    // Service control
    async serviceAction(action) {
        return this.request(`/api/service/${action}`, {
            method: 'POST',
        });
    },
};
