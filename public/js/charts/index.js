import ChartFactory from './factory.js';

export function initializeCharts() {
    const chartElements = document.querySelectorAll('[data-chart]');
    
    chartElements.forEach(element => {
        const historyData = element.getAttribute('data-chart-history');
        const regularData = element.getAttribute('data-chart-data');
        const options = element.getAttribute('data-chart-options');
        
        try {
            let data = [];
            let chartOptions = {};
            
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
            } else if (regularData) {
                data = JSON.parse(regularData);
            }
            
            if (options) {
                chartOptions = {
                    ...chartOptions,
                    ...JSON.parse(options)
                };
            }

            const type = element.getAttribute('data-chart-type') || 'bar';
            const chart = ChartFactory.create(type, data, chartOptions);
            element.innerHTML = chart.render();
        } catch (e) {
            console.error('Failed to create chart:', e);
            element.textContent = element.textContent;
        }
    });
}

export { default as ChartFactory } from './factory.js';