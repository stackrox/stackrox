import React from 'react';
import { Sunburst, DiscreteColorLegend, LabelSeries } from 'react-vis';
import PropTypes from 'prop-types';
import merge from 'deepmerge';

import SunburstDetailSection from 'Components/visuals/SunburstDetailSection';

// Get array of node ancestor names
function getKeyPath(node) {
    const name = node.name || node.data.name;
    if (!node.parent) {
        return [name];
    }

    return [name].concat(getKeyPath(node.parent));
}

// Update a dataset to highlight a specific set of nodes
function highlightPathData(data, highlightedNames) {
    if (data.children) {
        data.children.map(child => highlightPathData(child, highlightedNames));
    }

    /* eslint-disable */
    data.style = {
        ...data.style,
        fillOpacity: highlightedNames && !highlightedNames.includes(data.name) ? 0.5 : 1
    };
    /* eslint-enable */
    return data;
}

const LABEL_STYLE = {
    fontSize: '12px',
    textAnchor: 'middle'
};

export default class BasicSunburst extends React.Component {
    static propTypes = {
        data: PropTypes.shape({}).isRequired,
        legendData: PropTypes.arrayOf(PropTypes.object).isRequired,
        sunburstProps: PropTypes.shape({}),
        onValueMouseOver: PropTypes.func,
        onValueMouseOut: PropTypes.func,
        onValueSelect: PropTypes.func,
        onValueDeselect: PropTypes.func
    };

    static defaultProps = {
        sunburstProps: {},
        onValueMouseOver: null,
        onValueMouseOut: null,
        onValueSelect: null,
        onValueDeselect: null
    };

    constructor(props) {
        super(props);
        this.state = {
            data: this.props.data,
            clicked: false,
            selectedDatum: null
        };
    }

    getCenterLabel = () => {
        const { selectedDatum } = this.state;
        if (!selectedDatum) return null;
        return <LabelSeries data={[{ x: 0, y: 10, label: '76%', style: LABEL_STYLE }]} />;
    };

    onValueMouseOverHandler = datum => {
        const { data, clicked } = this.state;
        const { onValueMouseOver } = this.props;
        if (clicked) {
            return;
        }
        const path = getKeyPath(datum);
        this.setState({
            data: highlightPathData(data, path),
            selectedDatum: datum
        });
        if (onValueMouseOver) onValueMouseOver(path);
    };

    onValueMouseOutHandler = () => {
        const { data, clicked } = this.state;
        const { onValueMouseOut } = this.props;
        if (clicked) {
            return;
        }
        this.setState({
            selectedDatum: null,
            data: highlightPathData(data, false)
        });
        if (onValueMouseOut) onValueMouseOut();
    };

    onValueClickHandler = datum => {
        const { clicked } = this.state;
        const { onValueSelect, onValueDeselect } = this.props;
        this.setState({ clicked: !clicked });
        if (clicked && onValueSelect) {
            onValueSelect(datum);
        }
        if (!clicked && onValueDeselect) {
            onValueDeselect(datum);
        }
    };

    getSunburstProps = () => {
        const defaultSunburstProps = {
            colorType: 'literal',
            width: 275,
            height: 250,
            className: 'self-start',
            onValueMouseOver: this.onValueMouseOverHandler,
            onValueMouseOut: this.onValueMouseOutHandler,
            onValueClick: this.onValueClickHandler
        };
        return merge(defaultSunburstProps, this.props.sunburstProps);
    };

    render() {
        const { legendData } = this.props;
        const { clicked, data, selectedDatum } = this.state;

        const sunburstProps = this.getSunburstProps();
        const sunburstStyle = Object.assign(
            {
                stroke: '#ddd',
                strokeOpacity: 0.3,
                strokeWidth: '0.5'
            },
            this.props.sunburstProps.style
        );
        sunburstProps.style = sunburstStyle;

        return (
            <>
                <div className="flex flex-col justify-between">
                    <Sunburst data={data} {...sunburstProps} hideRootNode>
                        {this.getCenterLabel()}
                    </Sunburst>
                    <DiscreteColorLegend
                        orientation="horizontal"
                        items={legendData.map(item => item.title)}
                        colors={legendData.map(item => item.color)}
                        className="w-full horizontal-bar-legend border-t border-base-300 h-7"
                    />
                </div>
                <SunburstDetailSection selectedDatum={selectedDatum} clicked={clicked} />
            </>
        );
    }
}
