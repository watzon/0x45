import { DataNormalizer } from './normalize.js';

class ValueChart {
    constructor(data, options = {}) {
        this.data = data;
        this.options = {
            normalizer: options.normalizer || {
                inputUnit: options.inputUnit || 'raw',
                outputUnit: options.outputUnit || 'raw',
                precision: options.precision,
                format: 'full'
            }
        };

        this.normalizer = new DataNormalizer(this.options.normalizer);
    }

    render() {
        const value = typeof this.data === 'object' ? this.data.value : this.data;
        const normalizedValue = this.normalizer.normalize(value);
        return `<span class="chart-value">${normalizedValue}</span>`;
    }
}

export default ValueChart;
