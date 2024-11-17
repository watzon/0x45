// Data normalization library for charts
// Handles unit conversions and formatting consistently

const UNITS = {
    // Storage units
    'B': { base: 'B', factor: 1 },
    'KiB': { base: 'B', factor: 1024 },
    'MiB': { base: 'B', factor: 1024 * 1024 },
    'GiB': { base: 'B', factor: 1024 * 1024 * 1024 },
    'TiB': { base: 'B', factor: 1024 * 1024 * 1024 * 1024 },
    
    // Metric units
    '': { base: '', factor: 1 },
    'K': { base: '', factor: 1000 },
    'M': { base: '', factor: 1000000 },
    'G': { base: '', factor: 1000000000 },
    'T': { base: '', factor: 1000000000000 },
    
    // Time units
    'ms': { base: 'ms', factor: 1 },
    's': { base: 'ms', factor: 1000 },
    'min': { base: 'ms', factor: 60 * 1000 },
    'h': { base: 'ms', factor: 60 * 60 * 1000 },
    
    // Percentage (special case)
    '%': { base: '%', factor: 1 }
};

// Unit families for auto-scaling
const UNIT_FAMILIES = {
    'B': ['B', 'KiB', 'MiB', 'GiB', 'TiB'],
    '': ['', 'K', 'M', 'G', 'T'],
    'ms': ['ms', 's', 'min', 'h'],
    '%': ['%']
};

// Default precision for different base units
const DEFAULT_PRECISION = {
    'B': 2,     // Storage units
    '': 1,      // Metric units
    'ms': 0,    // Time units
    '%': 1      // Percentage
};

class DataNormalizer {
    constructor(options = {}) {
        this.options = {
            inputUnit: options.inputUnit || '',     // Input unit (e.g., 'B', 'MiB', 'ms')
            outputUnit: options.outputUnit || '',   // Desired output unit (or 'auto')
            precision: options.precision,           // Number of decimal places (optional)
            format: options.format || 'full',       // 'value', 'full' (with unit), or 'object'
            threshold: options.threshold || 1024    // Threshold for auto-scaling (default: 1024)
        };

        // Validate input unit
        if (!UNITS[this.options.inputUnit]) {
            throw new Error(`Invalid input unit: ${this.options.inputUnit}`);
        }

        // Handle 'auto' output unit
        if (this.options.outputUnit === 'auto') {
            // Will be determined during normalization
        } else if (!UNITS[this.options.outputUnit]) {
            throw new Error(`Invalid output unit: ${this.options.outputUnit}`);
        } else {
            // Validate unit compatibility for non-auto output
            const inputBase = UNITS[this.options.inputUnit].base;
            const outputBase = UNITS[this.options.outputUnit].base;
            if (inputBase !== outputBase) {
                throw new Error(`Incompatible units: ${this.options.inputUnit} cannot be converted to ${this.options.outputUnit}`);
            }
        }
    }

    normalize(value, format = null) {
        if (typeof value !== 'number') {
            return value;
        }

        const inputUnit = UNITS[this.options.inputUnit];
        const baseValue = value * inputUnit.factor;

        // Handle auto unit selection
        let outputUnit;
        if (this.options.outputUnit === 'auto') {
            const unitFamily = UNIT_FAMILIES[inputUnit.base];
            outputUnit = this.selectAppropriateUnit(baseValue, unitFamily);
        } else {
            outputUnit = UNITS[this.options.outputUnit];
        }

        const normalizedValue = baseValue / outputUnit.factor;
        return this.format(normalizedValue, outputUnit.factor === 1 ? '' : outputUnit.name, format);
    }

    selectAppropriateUnit(baseValue, unitFamily) {
        let selectedUnit = UNITS[unitFamily[0]];
        for (const unitName of unitFamily) {
            const unit = UNITS[unitName];
            if (baseValue >= unit.factor && baseValue >= this.options.threshold) {
                selectedUnit = { ...unit, name: unitName };
            }
        }
        return selectedUnit;
    }

    format(value, unit, overrideFormat = null) {
        // For 'auto' output, we get the base unit from the input unit
        const baseUnit = this.options.outputUnit === 'auto' 
            ? UNITS[this.options.inputUnit].base 
            : UNITS[this.options.outputUnit].base;

        const precision = this.options.precision ?? DEFAULT_PRECISION[baseUnit] ?? 0;
        const formattedValue = Number(value).toFixed(precision);
        const format = overrideFormat || this.options.format;

        // Get unit to display (use current unit for zero values)
        const displayUnit = value === 0 && this.options.outputUnit !== 'auto'
            ? this.options.outputUnit
            : unit;

        switch (format) {
            case 'full':
                return `${formattedValue} ${displayUnit}`.trim();
            case 'object':
                return {
                    value: Number(formattedValue),
                    unit: displayUnit
                };
            default:
                return Number(formattedValue);
        }
    }
}

// Helper function to automatically determine best unit
function autoUnit(value, baseUnit = '') {
    if (!baseUnit || !UNITS[baseUnit]) {
        return { value, unit: '' };
    }

    // Get all units with the same base
    const compatibleUnits = Object.entries(UNITS)
        .filter(([_, def]) => def.base === baseUnit)
        .sort((a, b) => a[1].factor - b[1].factor);

    // Find the largest unit that doesn't make the value < 1
    let unit = compatibleUnits[0][0];
    let factor = compatibleUnits[0][1].factor;

    for (const [unitName, unitDef] of compatibleUnits) {
        if (value >= unitDef.factor) {
            unit = unitName;
            factor = unitDef.factor;
        } else {
            break;
        }
    }

    return {
        value: value / factor,
        unit
    };
}

export { DataNormalizer, UNITS, autoUnit };
