function showToast(message) {
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

function copyToClipboard(text, element) {
    if (navigator.clipboard && window.isSecureContext) {
        navigator.clipboard.writeText(text)
            .then(() => showToast('Copied to clipboard'))
            .catch(err => {
                console.error('Failed to copy text: ', err);
            });
        return;
    }

    // Fallback for older browsers
    const textArea = document.createElement('textarea');
    textArea.value = text;
    
    // Avoid scrolling to bottom
    textArea.style.top = '0';
    textArea.style.left = '0';
    textArea.style.position = 'fixed';
    textArea.style.opacity = '0';
    
    document.body.appendChild(textArea);
    textArea.focus();
    textArea.select();

    try {
        document.execCommand('copy');
        showToast('Copied to clipboard');
    } catch (err) {
        console.error('Failed to copy text: ', err);
    }

    document.body.removeChild(textArea);
}

// Add click event listeners to all elements with data-clipboard attribute
document.addEventListener('DOMContentLoaded', () => {
    const clipboardElements = document.querySelectorAll('[data-clipboard]');
    
    clipboardElements.forEach(element => {
        element.addEventListener('click', (e) => {
            const selector = element.getAttribute('data-clipboard');
            let textToCopy;
            
            if (selector) {
                // If selector is provided, find the target element
                const target = document.querySelector(selector);
                textToCopy = target ? target.textContent : '';
            } else {
                // If no selector, use the data-content attribute or element's text
                textToCopy = element.getAttribute('data-content') || element.textContent;
            }
            
            copyToClipboard(textToCopy.trim(), element);
        });
    });
});
