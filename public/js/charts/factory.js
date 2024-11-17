import BarChart from './bar.js';
import PieChart from './pie.js';
import DotChart from './dot.js';
import ValueChart from './value.js';

class ChartFactory {
    static create(type, data, options) {
        switch (type) {
            case 'bar':
                return new BarChart(data, options);
            case 'pie':
                return new PieChart(data, options);
            case 'dot':
                return new DotChart(data, options);
            case 'value':
                return new ValueChart(data, options);
            default:
                throw new Error(`Unknown chart type: ${type}`);
        }
    }
}

export default ChartFactory;