function createAsciiChart(data, options = {}) {
    const {
        barWidth = 12,
        height = 5,
        color = 'blue',
        showLabels = true,
        showDates = true,
        barChar = '█',
        emptyChar = '░',
        showScale = true
    } = options;

    // Convert values to numbers for scaling
    const values = data.map(d => {
        if (typeof d.value === 'number') return d.value;
        if (typeof d.value === 'string') {
            const num = parseInt(d.value.replace(/[^0-9]/g, ''));
            return isNaN(num) ? 0 : num;
        }
        return 0;
    });

    const max = Math.max(...values);
    const min = Math.min(...values);
    const valueRange = max - min;

    // Helper function to calculate relative intensity
    const getIntensity = (value) => {
        if (valueRange === 0) return value > 0 ? 1 : 0;
        return (value - min) / valueRange;
    };

    // Format numbers with consistent width
    const formattedNumbers = data.map(d => {
        if (typeof d.value === 'string' && d.value.includes('B')) {
            return d.value.padStart(7);
        }
        return d.value.toString().padStart(3);
    });
    const maxNumberWidth = Math.max(...formattedNumbers.map(n => n.length));

    // Create the chart rows
    let rows = [];

    // Add value labels if requested
    if (showLabels) {
        const labels = formattedNumbers.map((num, i) => {
            const padding = Math.max(0, barWidth - num.length);
            const leftPad = Math.floor(padding / 2);
            const rightPad = padding - leftPad;
            const intensity = getIntensity(values[i]);
            return `<span class="chart-label" style="--value-intensity: ${intensity}">${' '.repeat(leftPad) + num + ' '.repeat(rightPad)}</span>`;
        }).join('');
        rows.push(' '.repeat(showScale ? 6 : 0) + labels);
        rows.push(' '.repeat(showScale ? 6 : 0) + ' '.repeat(labels.length));
    }

    // Create bars with scale
    for (let row = height - 1; row >= 0; row--) {
        const rowContent = values.map((value, i) => {
            const filled = Math.round((value / max) * height);
            const char = row < filled ? barChar : emptyChar;
            const intensity = getIntensity(value);
            const barContent = char.repeat(barWidth - 1).padEnd(barWidth);
            return `<span class="chart-bar" style="--value-intensity: ${intensity}">${barContent}</span>`;
        }).join('');

        if (showScale) {
            const scaleValue = max > 0 ? Math.round((max / height) * (height - row)) : 0;
            const scaleStr = (row === height - 1 ? 0 : scaleValue).toString().padStart(4);
            rows.push(`${scaleStr} │ ${rowContent}`);
        } else {
            rows.push(rowContent);
        }
    }

    // Add date labels if requested
    if (showDates) {
        const dates = data.map(d => {
            const date = new Date(d.date);
            const dateStr = date.toLocaleDateString('en-US', { 
                month: 'short', 
                day: 'numeric'
            });
            const padding = Math.max(0, barWidth - dateStr.length);
            const leftPad = Math.floor(padding / 2);
            const rightPad = padding - leftPad;
            return ' '.repeat(leftPad) + dateStr + ' '.repeat(rightPad);
        }).join('');
        rows.push(' '.repeat(showScale ? 6 : 0) + dates);
    }

    // Add summary row using existing values, min, and max
    const avg = Math.round(values.reduce((a, b) => a + b, 0) / values.length);
    
    // Add empty line before summary
    rows.push('');
    rows.push(' '.repeat(showScale ? 6 : 0) + `min: ${min} | avg: ${avg} | max: ${max}`);

    return `<pre class="chart" data-chart-color="${color}">${rows.join('\n')}</pre>`;
}

document.addEventListener('DOMContentLoaded', () => {
    const chartElements = document.querySelectorAll('[data-chart]');
    
    chartElements.forEach(element => {
        const historyData = element.getAttribute('data-chart-history');
        if (historyData) {
            try {
                const data = JSON.parse(historyData);
                const color = element.getAttribute('data-chart-color') || 'blue';
                const chart = createAsciiChart(data, {
                    color: color,
                    showLabels: true,
                    showDates: true,
                    showScale: true,
                    barWidth: 12,
                    height: 5
                });
                element.innerHTML = chart;
            } catch (e) {
                console.error('Failed to parse chart data:', e);
                const value = element.textContent;
                element.textContent = value;
            }
        }
    });
}); 