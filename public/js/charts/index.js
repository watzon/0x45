import ChartFactory from './factory.js';

export function initializeCharts() {
    const chartElements = document.querySelectorAll('[data-chart]');
    
    chartElements.forEach(element => {
        const historyData = element.getAttribute('data-chart-history');
        const options = element.getAttribute('data-chart-options');
        
        try {
            let data = [];
            let chartOptions = {};
            
            // Handle both history data and options data
            if (historyData) {
                data = JSON.parse(historyData);
                chartOptions = {
                    color: element.getAttribute('data-chart-color') || 'blue',
                    showLabels: true,
                    showDates: true,
                    showScale: true,
                    barWidth: 12,
                    height: 5
                };
            } else if (options) {
                // For dot charts with series data
                chartOptions = JSON.parse(options);
                data = []; // The data is contained within the series in options
            }

            const type = element.getAttribute('data-chart-type') || 'bar';
            console.log('Creating chart:', { type, options: chartOptions }); // Debug log
            
            const chart = ChartFactory.create(type, data, chartOptions);
            element.innerHTML = chart.render();
        } catch (e) {
            console.error('Failed to create chart:', e);
            element.textContent = element.textContent;
        }
    });
}

export { default as ChartFactory } from './factory.js';