// Simple client-side router using htmx
window.Router = {
    protectedPaths: [
        '/dashboard', 
        '/members',
        '/member',
        '/member/new',
        '/member/bulk',
        '/account',
        '/member'  // This will match all /member/* paths
    ],
    
    init() {
        this.loadCurrentPage();
        this.setupEventListeners();
    },
    
    isAuthenticated() {
        return !!localStorage.getItem('accessToken');
    },
    
    loadPage(path) {
        
        
        const templatePath = `/templates/pages${path === '/' ? '/home' : path}.html`;
        
        if (this.protectedPaths.some(p => path.startsWith(p)) && !this.isAuthenticated()) {
            console.log('Protected route accessed without auth, redirecting to login');
            this.navigate('/login');
            return;
        }
        
        htmx.ajax('GET', templatePath, {
            target: '#content',
            swap: 'innerHTML'
        });
    },
    
    navigate(path) {
        window.history.pushState({}, '', path);
        this.loadPage(path);
    },
    
    loadCurrentPage() {
        this.loadPage(window.location.pathname);
    },
    
    setupEventListeners() {
        window.addEventListener('popstate', () => this.loadCurrentPage());
    }
};

// Initialize router when DOM is ready
document.addEventListener('DOMContentLoaded', () => Router.init()); 