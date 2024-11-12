class Legend {
    constructor(options = {}) {
        this.options = {
            position: options.position || 'bottom', // 'bottom', 'right', 'top', 'left'
            dotChar: options.legendDotChar || '■',
            spacing: options.legendSpacing || 3,
            show: options.showLegend !== undefined ? options.showLegend : true
        };
    }

    render(items) {
        if (!this.options.show || !items?.length) return [];

        const legendItems = items.map(item => 
            `<span class="chart-legend-item" data-palette="${item.color}">${this.options.dotChar}</span> ${item.text}`
        );

        if (this.options.position === 'right') {
            return legendItems;
        }

        // For top, bottom, left positions, join items horizontally
        return [legendItems.join(' '.repeat(this.options.spacing))];
    }

    // Helper to wrap the rendered legend in appropriate container
    wrapOutput(content) {
        return `<div class="chart-legend chart-legend-${this.options.position}">${content}</div>`;
    }
}

export default Legend; 