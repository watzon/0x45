import { initializeClipboard } from './clipboard.js';
import { initializeCharts } from './charts/index.js';
import { initializeFileUpload } from './upload.js';

// Initialize all features when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    initializeClipboard();
    initializeCharts();
    initializeFileUpload();

    // Handle code area expansion
    const expandBtn = document.querySelector('.expand-btn');
    const pasteContent = document.querySelector('.paste-content');
    
    if (expandBtn && pasteContent) {
        expandBtn.addEventListener('click', () => {
            pasteContent.classList.toggle('expanded');
            expandBtn.textContent = pasteContent.classList.contains('expanded') ? 'collapse' : 'expand';
        });

        // Allow escape key to collapse
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape' && pasteContent.classList.contains('expanded')) {
                pasteContent.classList.remove('expanded');
                expandBtn.textContent = 'expand';
            }
        });
    }
});
