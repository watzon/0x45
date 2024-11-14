import Chart from './base.js';

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
            color: options.color || 'blue',
            valueFormat: options.valueFormat || (value => value.toString()),
            dateFormat: options.dateFormat || {
                month: 'short',
                day: 'numeric'
            }
        };

        super(data, mergedOptions);

        // Process values with data sampling if needed
        this.processDataSampling();

        this.formattedNumbers = this.data.map(d => {
            if (typeof d.value === 'string' && d.value.includes('B')) {
                return d.value.padStart(7);
            }
            return this.options.valueFormat(d.value).padStart(3);
        });
        this.maxNumberWidth = Math.max(...this.formattedNumbers.map(n => n.length));
    }

    processDataSampling() {
        if (typeof window === 'undefined') return;

        // Calculate maximum bars that can fit on screen
        // Account for scale (6 chars), spacing between bars (1 char), and some padding (40px)
        const scaleWidth = this.options.showScale ? 6 : 0;
        const maxBars = Math.floor((window.innerWidth - 40 - scaleWidth) / (this.options.barWidth + 1));

        if (this.values.length > maxBars) {
            // Sample data points to fit screen
            const step = Math.ceil(this.values.length / maxBars);
            this.values = this.values.filter((_, i) => i % step === 0).slice(0, maxBars);
            this.data = this.data.filter((_, i) => i % step === 0).slice(0, maxBars);
            
            // Recalculate min/max after sampling
            this.max = Math.max(...this.values);
            this.min = Math.min(...this.values);
            this.valueRange = this.max - this.min;
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
            const intensity = this.getIntensity(this.values[i]);
            
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
        
        const emptyLine = ' '.repeat(this.options.showScale ? 6 : 0) + 
            ' '.repeat((this.formattedNumbers.length * this.options.barWidth) + (this.formattedNumbers.length - 1));
        
        return [
            ' '.repeat(this.options.showScale ? 6 : 0) + labels,
            emptyLine
        ];
    }

    renderBars() {
        const rows = [];
        for (let row = this.options.height - 1; row >= 0; row--) {
            rows.push(this.renderBarRow(row));
        }
        return rows;
    }

    renderBarRow(row) {
        let rowContent = '';
        for (let i = 0; i < this.values.length; i++) {
            const value = this.values[i];
            const filled = Math.round((value / this.max) * this.options.height);
            const char = row < filled ? this.options.barChar : this.options.emptyChar;
            const intensity = this.getIntensity(value);
            
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
            const scaleValue = this.max > 0 ? Math.round((this.max / this.options.height) * (this.options.height - row)) : 0;
            const scaleStr = (row === this.options.height - 1 ? 0 : scaleValue).toString().padStart(4);
            return `${scaleStr} │ ${rowContent}`;
        }
        return rowContent;
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
        return [' '.repeat(this.options.showScale ? 6 : 0) + dates];
    }

    renderSummary() {
        const avg = Math.round(this.values.reduce((a, b) => a + b, 0) / this.values.length);
        return [
            '',
            ' '.repeat(this.options.showScale ? 6 : 0) + `min: ${this.min} | avg: ${avg} | max: ${this.max}`
        ];
    }

    wrapOutput(rows) {
        return `<pre class="chart" data-palette="${this.options.color}">${rows.join('\n')}</pre>`;
    }
}

export default AsciiBarChart; 