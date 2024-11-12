import Chart from './base.js';

class AsciiBarChart extends Chart {
    constructor(data, options = {}) {
        const mergedOptions = {
            barChar: options.barChar || '█',
            emptyChar: options.emptyChar || '░',
            height: options.height || 5,
            barWidth: options.barWidth || 12,
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

        this.formattedNumbers = this.data.map(d => {
            if (typeof d.value === 'string' && d.value.includes('B')) {
                return d.value.padStart(7);
            }
            return this.options.valueFormat(d.value).padStart(3);
        });
        this.maxNumberWidth = Math.max(...this.formattedNumbers.map(n => n.length));
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
        const labels = this.formattedNumbers.map((num, i) => {
            const padding = Math.max(0, this.options.barWidth - num.length);
            const leftPad = Math.floor(padding / 2);
            const rightPad = padding - leftPad;
            const intensity = this.getIntensity(this.values[i]);
            return `<span class="chart-label" style="--value-intensity: ${intensity}">${' '.repeat(leftPad) + num + ' '.repeat(rightPad)}</span>`;
        }).join('');
        return [
            ' '.repeat(this.options.showScale ? 6 : 0) + labels,
            ' '.repeat(this.options.showScale ? 6 : 0) + ' '.repeat(labels.length)
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
        const rowContent = this.values.map((value, i) => {
            const filled = Math.round((value / this.max) * this.options.height);
            const char = row < filled ? this.options.barChar : this.options.emptyChar;
            const intensity = this.getIntensity(value);
            const barContent = char.repeat(this.options.barWidth - 1).padEnd(this.options.barWidth);
            return `<span class="chart-bar" style="--value-intensity: ${intensity}">${barContent}</span>`;
        }).join('');

        if (this.options.showScale) {
            const scaleValue = this.max > 0 ? Math.round((this.max / this.options.height) * (this.options.height - row)) : 0;
            const scaleStr = (row === this.options.height - 1 ? 0 : scaleValue).toString().padStart(4);
            return `${scaleStr} │ ${rowContent}`;
        }
        return rowContent;
    }

    renderDates() {
        const dates = this.data.map(d => {
            const date = new Date(d.date);
            const dateStr = date.toLocaleDateString('en-US', this.options.dateFormat);
            const padding = Math.max(0, this.options.barWidth - dateStr.length);
            const leftPad = Math.floor(padding / 2);
            const rightPad = padding - leftPad;
            return ' '.repeat(leftPad) + dateStr + ' '.repeat(rightPad);
        }).join('');
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