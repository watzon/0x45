import { showToast } from './toast.js';

export function copyToClipboard(text, element) {
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

export function initializeClipboard() {
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
} 