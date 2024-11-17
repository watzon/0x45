import Chart from './base.js';
import { DataNormalizer } from './normalize.js';

class AsciiBarChart extends Chart {
    constructor(data, options = {}) {
        // Calculate responsive bar width
        const defaultBarWidth = typeof window !== 'undefined' && window.innerWidth < 480 ? 8 : 12;
        
        const mergedOptions = {
            barChar: options.barChar || '█',
            emptyChar: options.emptyChar || '░',
            height: options.height || 5,
            barWidth: options.barWidth || defaultBarWidth,
            showLabels: options.showLabels !== undefined ? options.showLabels : true,
            showDates: options.showDates !== undefined ? options.showDates : true,
            showScale: options.showScale !== undefined ? options.showScale : true,
            scalePosition: options.scalePosition || 'left', // 'left' or 'right'
            color: options.color || 'blue',
            dateFormat: options.dateFormat || {
                month: 'short',
                day: 'numeric'
            },
            // Data normalization options
            normalizer: options.normalizer || {
                inputUnit: options.inputUnit || 'raw',
                outputUnit: options.outputUnit || 'raw',
                precision: options.precision,
                format: 'value'  // Always use 'value' for chart, format in summary
            }
        };

        super(data, mergedOptions);
        
        // Create normalizer
        this.normalizer = new DataNormalizer(mergedOptions.normalizer);

        // Process values with data sampling if needed
        this.processDataSampling();

        // Store both raw and normalized values
        this.rawValues = this.values;
        this.normalizedValues = this.values.map(v => this.normalizer.normalize(v));

        // Calculate ranges and round max to match scale
        this.rawMin = Math.min(...this.rawValues);
        this.rawMax = Math.max(...this.rawValues);

        // First normalize the max value to get the proper unit scale
        const normalizedMax = Number(this.normalizer.normalize(this.rawMax, 'value'));
        
        // For unitless values (like counts), round to nice number
        if (!this.options.normalizer?.inputUnit) {
            const magnitude = Math.pow(10, Math.floor(Math.log10(normalizedMax)));
            this.scaledMax = Math.ceil(normalizedMax / magnitude) * magnitude;
        } else {
            // For values with units (like bytes), just round up
            this.scaledMax = Math.ceil(normalizedMax);
        }
        
        // Calculate unit based on max scale
        this.scaledUnit = this.scaledMax / this.options.height;

        // Format numbers for display
        this.formattedNumbers = this.values.map(v => {
            const normalizedValue = this.normalizer.normalize(v, 'full');
            return normalizedValue.toString().padStart(3);
        });
        this.maxNumberWidth = Math.max(...this.formattedNumbers.map(n => n.length));
    }

    processDataSampling() {
        if (typeof window === 'undefined') return;

        // Calculate maximum bars that can fit on screen
        const scaleWidth = this.options.showScale ? 6 : 0;
        const maxBars = Math.floor((window.innerWidth - 40 - scaleWidth) / (this.options.barWidth + 1));

        if (this.values.length > maxBars) {
            // Sample data points to fit screen
            const step = Math.ceil(this.values.length / maxBars);
            this.values = this.values.filter((_, i) => i % step === 0).slice(0, maxBars);
            this.data = this.data.filter((_, i) => i % step === 0).slice(0, maxBars);
        }
    }

    render() {
        const rows = [];
        if (this.options.showLabels) {
            rows.push(...this.renderLabels());
        }
        rows.push(...this.renderBars());
        if (this.options.showDates) {
            rows.push(...this.renderDates());
        }
        rows.push(...this.renderSummary());
        return this.wrapOutput(rows);
    }

    renderLabels() {
        let labels = '';
        for (let i = 0; i < this.formattedNumbers.length; i++) {
            const num = this.formattedNumbers[i];
            const barWidth = this.options.barWidth;
            const numWidth = num.length;
            
            // Calculate exact center position with a slight right offset
            const leftPad = Math.floor((barWidth - numWidth) / 2) + 1; 
            const rightPad = barWidth - numWidth - leftPad;
            const intensity = this.getIntensity(this.normalizedValues[i]);
            
            // Add left padding spaces
            for (let x = 0; x < leftPad; x++) {
                labels += `<span class="chart-label" style="--value-intensity: ${intensity}"> </span>`;
            }
            
            // Add each digit of the number
            for (let x = 0; x < num.length; x++) {
                labels += `<span class="chart-label" style="--value-intensity: ${intensity}">${num[x]}</span>`;
            }
            
            // Add right padding spaces
            for (let x = 0; x < rightPad; x++) {
                labels += `<span class="chart-label" style="--value-intensity: ${intensity}"> </span>`;
            }
            
            // Add spacing between bars (except for the last label)
            if (i < this.formattedNumbers.length - 1) {
                labels += `<span class="chart-spacer"> </span>`;
            }
        }
        
        const scaleWidth = this.options.showScale ? this.getMaxScaleWidth() + 3 : 0; // +3 for " │ "
        const scalePadding = ' '.repeat(scaleWidth);
        
        return [
            (this.options.scalePosition === 'left' ? scalePadding : '') + labels + (this.options.scalePosition === 'right' ? scalePadding : ''),
            (this.options.scalePosition === 'left' ? scalePadding : '') + ' '.repeat(labels.length) + (this.options.scalePosition === 'right' ? scalePadding : '')
        ];
    }

    renderBars() {
        const rows = [];
        // Render rows from top to bottom
        for (let row = 0; row < this.options.height; row++) {
            rows.push(this.renderBarRow(row));
        }
        return rows;
    }

    renderBarRow(row) {
        let rowContent = '';
        for (let i = 0; i < this.values.length; i++) {
            const rawValue = this.rawValues[i];
            const normalizedValue = Number(this.normalizer.normalize(rawValue, 'value'));
            const filled = Math.round((normalizedValue / this.scaledMax) * this.options.height);
            // Check against inverted row to fill from bottom up
            const char = filled > (this.options.height - row - 1) ? this.options.barChar : this.options.emptyChar;
            const intensity = normalizedValue / this.scaledMax;
            
            // Render each character individually
            for (let x = 0; x < this.options.barWidth; x++) {
                rowContent += `<span class="chart-bar" style="--value-intensity: ${intensity}">${char}</span>`;
            }
            
            // Add spacing between bars (except for the last bar)
            if (i < this.values.length - 1) {
                rowContent += `<span class="chart-spacer"> </span>`;
            }
        }

        if (this.options.showScale) {
            // Calculate scale value for current row (from bottom to top)
            // Invert row to make scale go from max at top to 0 at bottom
            const scaleValue = (this.options.height - row - 1) * this.scaledUnit;
            
            let valueToFormat;
            if (this.options.normalizer?.inputUnit) {
                // For values with units (like bytes), convert back
                const scaleBytes = Math.round(scaleValue * 1024 * 1024);
                valueToFormat = row === 0 ? this.rawMax : scaleBytes;
            } else {
                // For unitless values, use scale value directly
                valueToFormat = row === 0 ? this.rawMax : scaleValue;
            }
            
            const formattedScale = this.normalizer.normalize(valueToFormat, 'full');
            
            // Calculate padding based on the longest scale value
            const maxScaleWidth = this.getMaxScaleWidth();
            const paddedScale = formattedScale.padStart(maxScaleWidth);

            // Return row with scale on the specified side
            return this.options.scalePosition === 'left'
                ? `${paddedScale} │ ${rowContent}`
                : `${rowContent} │ ${paddedScale}`;
        }
        return rowContent;
    }

    getMaxScaleWidth() {
        const scaleValues = [];
        // Include the actual max value for proper width calculation
        scaleValues.push(this.normalizer.normalize(this.rawMax, 'full').length);
        
        for (let i = 0; i < this.options.height; i++) {
            const value = i * this.scaledUnit;
            let valueToFormat;
            if (this.options.normalizer?.inputUnit) {
                const scaleBytes = Math.round(value * 1024 * 1024);
                valueToFormat = scaleBytes;
            } else {
                valueToFormat = value;
            }
            const formatted = this.normalizer.normalize(valueToFormat, 'full');
            scaleValues.push(formatted.length);
        }
        return Math.max(...scaleValues);
    }

    getIntensity(value) {
        if (this.rawMax === 0) return value > 0 ? 1 : 0;
        const normalizedValue = Number(this.normalizer.normalize(value, 'value'));
        return normalizedValue / this.scaledMax;
    }

    renderDates() {
        let dates = '';
        for (let i = 0; i < this.data.length; i++) {
            const date = new Date(this.data[i].date);
            const dateStr = date.toLocaleDateString('en-US', this.options.dateFormat);
            const barWidth = this.options.barWidth;
            const dateWidth = dateStr.length;
            
            // Calculate exact center position with extra left padding
            const leftPad = Math.floor((barWidth - dateWidth) / 2) + 1; 
            const rightPad = barWidth - dateWidth - leftPad;
            
            // Add left padding
            dates += ' '.repeat(leftPad);
            // Add the date string
            dates += dateStr;
            // Add right padding
            dates += ' '.repeat(rightPad);
            
            // Add spacing between dates (except for the last date)
            if (i < this.data.length - 1) {
                dates += ' ';
            }
        }

        const scaleWidth = this.options.showScale ? this.getMaxScaleWidth() + 3 : 0;
        return [(this.options.scalePosition === 'left' ? ' '.repeat(scaleWidth) : '') + dates + (this.options.scalePosition === 'right' ? ' '.repeat(scaleWidth) : '')];
    }

    renderSummary() {
        const min = this.normalizer.normalize(this.rawMin, 'full');
        const max = this.normalizer.normalize(this.rawMax, 'full');
        const avg = this.normalizer.normalize(
            Math.round(this.rawValues.reduce((a, b) => a + b, 0) / this.rawValues.length),
            'full'
        );

        const scaleWidth = this.options.showScale ? this.getMaxScaleWidth() + 3 : 0;
        return [
            '',
            (this.options.scalePosition === 'left' ? ' '.repeat(scaleWidth) : '') + 
            `min: ${min} | avg: ${avg} | max: ${max}` +
            (this.options.scalePosition === 'right' ? ' '.repeat(scaleWidth) : '')
        ];
    }

    wrapOutput(rows) {
        return `<pre class="chart" data-palette="${this.options.color}">${rows.join('\n')}</pre>`;
    }
}

export default AsciiBarChart;