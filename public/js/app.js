import { initializeClipboard } from './clipboard.js';
import { initializeCharts } from './charts/index.js';

// Initialize all features when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    initializeClipboard();
    initializeCharts();
});
