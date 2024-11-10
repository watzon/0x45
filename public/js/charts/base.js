// Base chart class with common functionality
class Chart {
    constructor(data, options = {}) {
        this.data = data;
        this.options = {
            barWidth: 12,
            height: 5,
            color: 'blue',
            showLabels: true,
            showDates: true,
            showScale: true,
            ...options
        };
        
        // Process values once during initialization
        this.values = this.processValues();
        this.max = Math.max(...this.values);
        this.min = Math.min(...this.values);
        this.valueRange = this.max - this.min;
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

    getIntensity(value) {
        if (this.valueRange === 0) return value > 0 ? 1 : 0;
        return (value - this.min) / this.valueRange;
    }

    render() {
        throw new Error('render() must be implemented by child classes');
    }
}

export default Chart; 