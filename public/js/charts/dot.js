import Chart from './base.js';

class AsciiDotChart extends Chart {
    constructor(data, options = {}) {
        // First, create default formatters
        const defaultFormatters = {
            xAxis: {
                format: (value) => `${value} MiB`
            },
            yAxis: {
                format: (value) => Math.round(value).toString()
            }
        };

        // Merge options, being careful not to override formatters unless explicitly provided
        const mergedOptions = {
            ...options, // User options come first as base
            
            // Then we layer our defaults on top for any missing values
            dotChar: options.dotChar || '•',
            lineChar: options.lineChar || '·',
            legendDotChar: options.legendDotChar || '■',
            scaleLineChar: options.scaleLineChar || '─',
            
            // Layout
            height: options.height || 15,
            barWidth: options.barWidth || 30,
            yAxisPadding: options.yAxisPadding || 8,
            legendSpacing: options.legendSpacing || 3,
            
            // Display options
            connectLines: options.connectLines !== undefined ? options.connectLines : true,
            showScale: options.showScale !== undefined ? options.showScale : true,
            showXScale: options.showXScale !== undefined ? options.showXScale : true,
            showYScale: options.showYScale !== undefined ? options.showYScale : true,
            showLegend: options.showLegend !== undefined ? options.showLegend : true,
            showAxisLabels: options.showAxisLabels !== undefined ? options.showAxisLabels : true,
            
            // Legend options
            legendPosition: options.legendPosition || 'bottom',
            
            // Axis configuration
            xAxis: {
                label: "File Size",
                unit: "MiB",
                points: 3,
                maxSize: null,
                ...(options.xAxis || {}),
                format: (options.xAxis && options.xAxis.format) || defaultFormatters.xAxis.format
            },
            
            yAxis: {
                label: "Days",
                points: 3,
                range: {
                    min: null,
                    max: null,
                    ...((options.yAxis && options.yAxis.range) || {})
                },
                ...(options.yAxis || {}),
                format: (options.yAxis && options.yAxis.format) || defaultFormatters.yAxis.format
            },
            
            // Series styling
            series: options.series?.map(series => ({
                ...series,
                palette: series.palette || 'blue'
            })) || [],
            
            // Line styling
            lineOpacity: options.lineOpacity || 0.3,
            dotOpacity: options.dotOpacity || 1.0
        };

        super(data, mergedOptions);
        
        // Process series data with colors
        this.series = this.options.series.map(series => ({
            ...series,
            values: this.processValues(series.data)
        }));
        
        // Calculate ranges
        const allValues = this.series.flatMap(s => s.values);
        this.max = this.options.yAxis.range.max || Math.max(...allValues);
        this.min = this.options.yAxis.range.min || Math.min(...allValues);
        this.valueRange = this.max - this.min;
    }

    addYScale(grid) {
        if (!this.options.showYScale) return;

        const height = this.options.height * 2;
        const scalePoints = [];
        const step = (this.max - this.min) / (this.options.yAxis.points - 1);
        
        for (let i = 0; i < this.options.yAxis.points; i++) {
            scalePoints.push(this.max - (step * i));
        }
        
        const padding = this.options.yAxisPadding;
        
        // Add y-axis label if enabled
        if (this.options.showAxisLabels) {
            const yLabel = this.options.yAxis.label;
            const yLabelY = Math.floor(height / 2);
            for (let i = 0; i < yLabel.length; i++) {
                grid[yLabelY + i][0] = yLabel[i];
            }
        }
        
        scalePoints.forEach((value, index) => {
            const y = Math.floor((index / (this.options.yAxis.points - 1)) * (height - 1));
            const label = this.options.yAxis.format(value).padStart(6);
            
            for (let x = 0; x < 6; x++) {
                grid[y][x + 1] = label[x];
            }
            grid[y][7] = this.options.scaleLineChar;
        });
    }

    addXScale(width) {
        if (!this.options.showXScale) return [];

        const padding = this.options.yAxisPadding;
        const rows = [];
        const points = this.options.xAxis.points;
        const scalePoints = Array.from({length: points}, (_, i) => 
            Math.floor((i / (points - 1)) * (width - 1)));
        
        // Parse maxSize
        let maxSize = this.parseSize(this.options.xAxis.maxSize);
        
        // Create scale line
        let scaleLine = ' '.repeat(padding);
        scalePoints.forEach(x => {
            const value = Math.round((x / (width - 1)) * maxSize);
            const label = this.options.xAxis.format(value);
            const position = Math.floor(x + padding);
            scaleLine += label.padStart(position - scaleLine.length + label.length);
        });
        
        rows.push(scaleLine);
        
        // Add x-axis label if enabled
        if (this.options.showAxisLabels) {
            rows.push('');
            const xLabel = this.options.xAxis.label;
            const xLabelPadding = Math.floor((width + padding - xLabel.length) / 2);
            rows.push(' '.repeat(xLabelPadding) + xLabel);
        }
        
        return rows;
    }

    parseSize(size) {
        if (typeof size === 'number') return size;
        if (typeof size === 'string') {
            const match = size.match(/(\d+)\s*([KMGT]i?B)?/i);
            if (match) {
                const num = parseInt(match[1]);
                const unit = match[2]?.toUpperCase() || 'B';
                const units = {
                    'B': 1,
                    'KIB': 1024,
                    'MIB': 1024*1024,
                    'GIB': 1024*1024*1024,
                    'TIB': 1024*1024*1024*1024
                };
                return num * (units[unit] || 1) / (1024 * 1024);
            }
        }
        return 512; // Default fallback
    }

    renderLegend() {
        if (!this.options.showLegend) return [];
        
        const legend = [];
        const legendItems = this.series.map(series => 
            `<span class="chart-legend-dot" data-palette="${series.palette}">${this.options.legendDotChar}</span> ${series.name}`
        );
        legend.push(legendItems.join(' '.repeat(this.options.legendSpacing)));
        return legend;
    }

    drawSeriesLine(grid, series) {
        const width = this.options.barWidth * 2;
        const height = this.options.height * 2;
        const data = series.data;
        const padding = this.options.yAxisPadding;
        
        let lastX = null;
        let lastY = null;
        
        data.forEach((point, index) => {
            const x = Math.floor((index / (data.length - 1)) * (width - 1)) + padding;
            const normalizedValue = (point.value - this.min) / (this.max - this.min);
            const y = Math.floor((1 - normalizedValue) * (height - 1));
            
            if (y >= 0 && y < height && x >= 0 && x < grid[0].length) {
                // Update to use palette
                grid[y][x] = `<span class="chart-dot" data-palette="${series.palette}">${this.options.dotChar}</span>`;
                
                if (this.options.connectLines && lastX !== null) {
                    this.drawLine(grid, lastX, lastY, x, y, series.palette);
                }
                
                lastX = x;
                lastY = y;
            }
        });
    }

    // Override processValues to handle series data
    processValues(data) {
        if (!Array.isArray(data)) {
            console.warn('Expected array for data, got:', data);
            return [];
        }
        return data.map(point => {
            if (typeof point === 'object' && point !== null) {
                return typeof point.value === 'number' ? point.value : 0;
            }
            return typeof point === 'number' ? point : 0;
        });
    }

    render() {
        const rows = [];
        const width = this.options.barWidth * 2;
        const height = this.options.height * 2;
        
        // Create the chart grid with extra padding for y-axis
        const grid = Array(height).fill().map(() => 
            Array(width + this.options.yAxisPadding).fill(' ')
        );
        
        // Draw each series
        this.series.forEach(series => {
            this.drawSeriesLine(grid, series);
        });
        
        // Add y-axis scale if requested
        if (this.options.showScale) {
            this.addYScale(grid);
        }
        
        // Convert grid to string rows
        rows.push(...grid.map(row => row.join('')));
        
        // Add x-axis scale if requested
        if (this.options.showXScale) {
            rows.push(...this.addXScale(width));
        }
        
        // Add legend if multiple series
        if (this.series.length > 1) {
            rows.push('');
            rows.push(...this.renderLegend());
        }
        
        return this.wrapOutput(rows);
    }

    drawLine(grid, x1, y1, x2, y2, palette) {
        // Bresenham's line algorithm
        const dx = Math.abs(x2 - x1);
        const dy = Math.abs(y2 - y1);
        const sx = x1 < x2 ? 1 : -1;
        const sy = y1 < y2 ? 1 : -1;
        let err = dx - dy;
        
        let currentX = x1;
        let currentY = y1;
        
        while (true) {
            // Only draw if within bounds
            if (currentY >= 0 && currentY < grid.length && 
                currentX >= 0 && currentX < grid[0].length) {
                // Only draw if the cell is empty
                if (grid[currentY][currentX] === ' ') {
                    // Update to use palette
                    grid[currentY][currentX] = `<span class="chart-line" data-palette="${palette}" style="opacity: ${this.options.lineOpacity}">${this.options.lineChar}</span>`;
                }
            }
            
            if (currentX === x2 && currentY === y2) break;
            
            const e2 = 2 * err;
            if (e2 > -dy) {
                err -= dy;
                currentX += sx;
            }
            if (e2 < dx) {
                err += dx;
                currentY += sy;
            }
        }
    }

    wrapOutput(rows) {
        return `<pre class="chart">${rows.join('\n')}</pre>`;
    }
}

export default AsciiDotChart; 