function showCopyNotification(element) {
    const notification = document.createElement('div');
    notification.textContent = 'Copied!';
    notification.className = 'copy-notification';
    element.appendChild(notification);
    
    // Remove the notification after animation
    setTimeout(() => {
        notification.remove();
    }, 1500);
}

function copyToClipboard(text, element) {
    if (navigator.clipboard && window.isSecureContext) {
        navigator.clipboard.writeText(text)
            .then(() => showCopyNotification(element))
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
    } catch (err) {
        console.error('Failed to copy text: ', err);
    }

    document.body.removeChild(textArea);

    showCopyNotification(element);
}

// Add click event listeners to all elements with data-clipboard attribute
document.addEventListener('DOMContentLoaded', () => {
    const clipboardElements = document.querySelectorAll('[data-clipboard]');
    
    clipboardElements.forEach(element => {
        element.addEventListener('click', (e) => {
            const text = e.target.textContent;
            copyToClipboard(text, e.target);
        });
    });
});
