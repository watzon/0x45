import Chart from './base.js';
import Legend from './legend.js';

class AsciiPieChart extends Chart {
    constructor(data, options = {}) {
        const mergedOptions = {
            width: options.width || 19,
            dotChar: '•',
            legendChar: '■',
            colors: ['pie1', 'pie2', 'pie3', 'pie4', 'pie5', 'pie6'],
            legendPosition: options.legendPosition || 'right'
        };

        // Transform object data into array format if needed
        const processedData = Array.isArray(data) 
            ? data 
            : Object.entries(data).map(([label, value]) => ({ label, value }));

        super(processedData, mergedOptions);
        
        this.total = this.values.reduce((sum, val) => sum + val, 0);
        
        // Handle empty or zero-sum data
        if (this.total === 0) {
            this.segments = [{
                label: 'No Data',
                value: 1,
                percentage: 100,
                startAngle: 0,
                endAngle: 360,
                color: this.options.colors[0]
            }];
        } else {
            this.segments = this.calculateSegments();
        }

        this.legend = new Legend({
            position: mergedOptions.legendPosition,
            legendDotChar: mergedOptions.legendChar,
            showLegend: options.showLegend
        });
    }

    processValues() {
        return this.data.map(d => {
            const val = typeof d === 'object' ? d.value : d;
            if (typeof val === 'number') return val;
            if (typeof val === 'string') {
                const num = parseFloat(val.replace(/[^0-9.-]/g, ''));
                return isNaN(num) ? 0 : num;
            }
            return 0;
        });
    }

    calculateSegments() {
        let startAngle = 0;
        return this.data.map((item, index) => {
            const percentage = (this.values[index] / this.total) * 100;
            const angleSize = (percentage / 100) * 360;
            const segment = {
                label: item.label,
                value: this.values[index],
                percentage,
                startAngle,
                endAngle: startAngle + angleSize,
                color: this.options.colors[index % this.options.colors.length]
            };
            startAngle += angleSize;
            return segment;
        });
    }

    render() {
        const pieRows = this.generatePieAscii();
        const legendRows = this.generateLegend();
        
        return this.wrapOutput(`
<div class="chart-container chart-container-${this.legend.options.position}">
    <div class="chart-pie">
${pieRows.join('\n')}
    </div>
    <div class="chart-legend">
${legendRows.join('\n')}
    </div>
</div>`);
    }

    generatePieAscii() {
        const rows = [];
        const radius = Math.floor(this.options.width / 2);
        const aspectRatio = 2.1;
        
        // Reduce the vertical range by adjusting the y-bounds
        for (let y = -Math.floor(radius/aspectRatio); y <= Math.floor(radius/aspectRatio); y++) {
            let row = '';
            for (let x = -radius; x <= radius; x++) {
                const normalizedX = x;
                const normalizedY = y * aspectRatio;
                
                const distance = Math.sqrt(normalizedX * normalizedX + normalizedY * normalizedY);
                
                if (distance <= radius) {
                    let angle = Math.atan2(normalizedY, normalizedX) * (180 / Math.PI);
                    angle = angle < 0 ? angle + 360 : angle;
                    
                    const segment = this.segments.find(seg => 
                        angle >= seg.startAngle && angle < seg.endAngle);
                    
                    row += `<span class="chart-pie-segment" data-palette="${segment.color}">${this.options.dotChar}</span>`;
                } else {
                    row += ' ';
                }
            }
            if (row.trim()) {  // Only add rows that have content
                rows.push(row);
            }
        }
        
        return rows;
    }

    generateLegend() {
        if (this.total === 0) {
            return this.legend.render([{
                color: this.options.colors[0],
                text: 'No data available'
            }]);
        }

        const legendItems = this.segments.map(segment => ({
            color: segment.color,
            text: `${segment.percentage.toFixed(1).padStart(5)}% [${this.formatBytes(segment.value)}] ${segment.label}`
        }));

        return this.legend.render(legendItems);
    }

    // Helper function to format bytes into human-readable format
    formatBytes(bytes) {
        if (bytes === 0) return '0 B';
        const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(1024));
        return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${sizes[i]}`;
    }

    wrapOutput(content) {
        return `<pre class="chart">${content}</pre>`;
    }
}

export default AsciiPieChart; 