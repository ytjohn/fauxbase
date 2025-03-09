// Auth service for managing tokens and authentication state
window.AuthService = {
    // Token storage keys
    ACCESS_TOKEN_KEY: 'accessToken',
    REFRESH_TOKEN_KEY: 'refreshToken',

    // Store tokens from login response
    setTokens(accessToken, refreshToken) {
        localStorage.setItem(this.ACCESS_TOKEN_KEY, accessToken);
        localStorage.setItem(this.REFRESH_TOKEN_KEY, refreshToken);
        // Dispatch event for UI updates
        document.dispatchEvent(new CustomEvent('auth:changed'));
    },

    // Clear tokens on logout
    clearTokens() {
        localStorage.removeItem(this.ACCESS_TOKEN_KEY);
        localStorage.removeItem(this.REFRESH_TOKEN_KEY);
        document.dispatchEvent(new CustomEvent('auth:changed'));
    },

    // Check if user is authenticated
    isAuthenticated() {
        return !!localStorage.getItem(this.ACCESS_TOKEN_KEY);
    },

    // Get access token for API requests
    getAccessToken() {
        return localStorage.getItem(this.ACCESS_TOKEN_KEY);
    },

    // Handle login
    async login(email, password) {
        try {
            const response = await fetch('/api/v1/auth/login', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    email: email,
                    password: password
                })
            });

            if (!response.ok) {
                throw new Error('Login failed');
            }

            const data = await response.json();
            this.setTokens(data.accessToken, data.refreshToken);
            return true;
        } catch (error) {
            console.error('Login error:', error);
            return false;
        }
    },

    // Handle logout
    async logout() {
        try {
            if (this.isAuthenticated()) {
                await fetch('/api/v1/auth/logout', {
                    method: 'POST',
                    headers: {
                        'Authorization': `Bearer ${this.getAccessToken()}`
                    }
                });
            }
        } catch (error) {
            console.error('Logout error:', error);
        } finally {
            this.clearTokens();
        }
    }
}; 