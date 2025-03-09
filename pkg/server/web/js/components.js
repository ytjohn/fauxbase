// Register Alpine.js components
document.addEventListener('alpine:init', () => {

    // Sidebar component
    Alpine.data('sidebar', () => ({
        publicMenuItems: [
            { href: '/', label: 'Home' },
            { href: '/about', label: 'About' }
        ],
        
        authMenuItems: [
            { href: '/account', label: 'Account' }
        ],
        
        isAuthenticated() {
            return AuthService.isAuthenticated();
        },
        
        isCurrentPath(path) {
            return window.location.pathname === path ||
                   (window.location.pathname === '/' && path === '/');
        },
        
        init() {
            // Listen for route changes to update active state
            window.addEventListener('popstate', () => {
                this.$nextTick(() => this.$forceUpdate());
            });
            
            // Listen for auth changes to show/hide menu items
            document.addEventListener('auth:changed', () => {
                this.$nextTick(() => this.$forceUpdate());
            });
        }
    }));

    // Add new app title component
    Alpine.data('appTitle', () => ({
        title: 'FauxBase', // Default value
        
        init() {
            // Try to get from cache first
            const cached = DataService.getCachedData('versionInfo');
            if (cached) {
                this.title = cached.appName;
                return;
            }
            
            // Otherwise fetch it
            this.fetchTitle();
        },
        
        async fetchTitle() {
            try {
                const response = await fetch('/api/v1/version');
                if (!response.ok) throw new Error('Failed to fetch version info');
                
                const data = await response.json();
                this.title = data.appName;
                
                // Cache the version info
                DataService.setCachedData('versionInfo', data);
            } catch (error) {
                console.error('Error fetching app title:', error);
            }
        }
    }));

    // Footer component that shows version info
    Alpine.data('footerInfo', () => ({
        loading: true,
        error: null,
        data: {
            appName: 'FauxBase', // Updated default
            copyright: 'Copyright 2024',
            version: ''
        },

        async init() {
            // Try to get from cache first
            const cached = DataService.getCachedData('versionInfo');
            if (cached) {
                this.data = {
                    ...this.data,
                    ...cached
                };
                this.loading = false;
                return;
            }

            try {
                const response = await fetch('/api/v1/version');
                if (!response.ok) throw new Error('Failed to fetch version info');
                
                const responseData = await response.json();
                this.data = {
                    ...this.data,
                    ...responseData
                };
                
                // Cache the version info
                DataService.setCachedData('versionInfo', responseData);
            } catch (error) {
                console.error('Error fetching version:', error);
                this.error = 'Failed to load version information';
            } finally {
                this.loading = false;
            }
        }
    }));

    // About page component
    Alpine.data('aboutPage', () => ({
        loading: true,
        error: null,
        info: null,
        
        init() {
            this.fetchVersionInfo();
        },
        
        async fetchVersionInfo() {
            try {
                const response = await fetch('/api/v1/version');
                if (!response.ok) throw new Error('Failed to fetch version info');
                
                this.info = await response.json();
            } catch (error) {
                console.error('Error fetching version:', error);
                this.error = 'Failed to load version information';
            } finally {
                this.loading = false;
            }
        }
    }));

    // Login form component
    Alpine.data('loginForm', () => ({
        email: '',
        password: '',
        loading: false,
        successMessage: '',
        errorMessage: '',
        errors: {},

        async login() {
            this.loading = true;
            this.errorMessage = '';
            this.errors = {};

            try {
                const success = await AuthService.login(this.email, this.password);
                if (success) {
                    this.successMessage = 'Login successful!';
                    window.location.href = '/account';
                } else {
                    this.errorMessage = 'Invalid email or password';
                }
            } catch (error) {
                console.error('Login error:', error);
                this.errorMessage = 'Login failed. Please try again.';
            } finally {
                this.loading = false;
            }
        }
    }));

    // Account page component
    Alpine.data('accountPage', () => ({
        user: null,
        loading: true,
        error: null,
        copySuccess: false,

        init() {
            this.fetchUserInfo();
        },

        async fetchUserInfo() {
            try {
                const response = await fetch('/api/v1/whoami', {
                    headers: {
                        'Authorization': `Bearer ${AuthService.getAccessToken()}`
                    }
                });

                if (!response.ok) {
                    throw new Error('Failed to fetch user info');
                }

                const data = await response.json();
                this.user = data;
            } catch (error) {
                console.error('Error fetching user info:', error);
                this.error = 'Failed to load user information';
            } finally {
                this.loading = false;
            }
        },

        async copyToken() {
            const token = AuthService.getAccessToken();
            try {
                await navigator.clipboard.writeText(token);
                this.copySuccess = true;
                setTimeout(() => {
                    this.copySuccess = false;
                }, 2000);
            } catch (err) {
                console.error('Failed to copy token:', err);
            }
        }
    }));

    // Header Auth component
    Alpine.data('headerAuth', () => ({
        dropdownOpen: false,
        loading: true,
        userDisplay: '',
        userInitial: '',

        init() {
            this.updateUserInfo();
            // Listen for auth changes
            document.addEventListener('auth:changed', () => this.updateUserInfo());
        },

        isAuthenticated() {
            return AuthService.isAuthenticated();
        },

        async updateUserInfo() {
            this.loading = true;
            if (this.isAuthenticated()) {
                try {
                    const response = await fetch('/api/v1/whoami', {
                        headers: {
                            'Authorization': `Bearer ${AuthService.getAccessToken()}`
                        }
                    });
                    if (response.ok) {
                        const data = await response.json();
                        this.userDisplay = data.body.email;
                        this.userInitial = data.body.email[0].toUpperCase();
                    }
                } catch (error) {
                    console.error('Error fetching user info:', error);
                }
            }
            this.loading = false;
        },

        async logout() {
            await AuthService.logout();
            window.location.href = '/';
        }
    }));
});

// Add this before Alpine.js component registration
window.DataService = {
    cache: new Map(),
    
    setCachedData(key, data) {
        this.cache.set(key, data);
        document.dispatchEvent(new CustomEvent(`data:${key}Updated`));
    },
    
    getCachedData(key) {
        return this.cache.get(key);
    },
    
    clearCache() {
        this.cache.clear();
    }
};