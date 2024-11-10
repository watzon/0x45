export function showToast(message) {
    // Create or get toast container
    let container = document.querySelector('.toast-container');
    if (!container) {
        container = document.createElement('div');
        container.className = 'toast-container';
        document.body.appendChild(container);
    }

    // Create toast
    const toast = document.createElement('div');
    toast.className = 'toast';
    toast.textContent = message;
    container.appendChild(toast);

    // Remove toast after animation
    setTimeout(() => {
        toast.remove();
        // Remove container if empty
        if (container.children.length === 0) {
            container.remove();
        }
    }, 1500);
} 