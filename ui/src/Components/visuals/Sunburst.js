import React from 'react';
import { Sunburst, DiscreteColorLegend } from 'react-vis';
import PropTypes from 'prop-types';

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

export default class BasicSunburst extends React.Component {
    static propTypes = {
        data: PropTypes.shape({}).isRequired,
        legendData: PropTypes.arrayOf(PropTypes.object).isRequired,
        containerProps: PropTypes.shape({}),
        sunburstProps: PropTypes.shape({}),
        onValueMouseOver: PropTypes.func,
        onValueMouseOut: PropTypes.func,
        onValueSelect: PropTypes.func,
        onValueDeselect: PropTypes.func
    };

    static defaultProps = {
        containerProps: {},
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
            clicked: false
        };
    }

    render() {
        const {
            legendData,
            onValueMouseOver,
            onValueMouseOut,
            onValueSelect,
            onValueDeselect
        } = this.props;
        const { clicked, data } = this.state;

        const defaultContainerProps = {
            className: 'flex flex-col justify-between h-full'
        };
        const defaultSunburstProps = {
            colorType: 'literal',
            width: 275,
            height: 250,
            className: 'self-start',
            onValueMouseOver: datum => {
                if (clicked) {
                    return;
                }
                const path = getKeyPath(datum);
                this.setState({
                    data: highlightPathData(data, path)
                });
                if (onValueMouseOver) onValueMouseOver(path);
            },
            onValueMouseOut: () => {
                if (clicked) {
                    return;
                }
                this.setState({
                    data: highlightPathData(data, false)
                });
                if (onValueMouseOut) onValueMouseOut();
            },
            onValueClick: datum => {
                const clickState = !clicked;
                this.setState({ clicked: clickState });
                if (clickState && onValueSelect) {
                    onValueSelect(datum);
                }
                if (!clickState && onValueDeselect) {
                    onValueDeselect(datum);
                }
            }
        };

        const sunburstStyle = Object.assign(
            {
                stroke: '#ddd',
                strokeOpacity: 0.3,
                strokeWidth: '0.5'
            },
            this.props.sunburstProps.style
        );

        const containerProps = Object.assign({}, defaultContainerProps, this.props.containerProps);
        const sunburstProps = Object.assign({}, defaultSunburstProps, this.props.sunburstProps);
        sunburstProps.style = sunburstStyle;

        return (
            <div {...containerProps}>
                <Sunburst data={data} {...sunburstProps} />
                <DiscreteColorLegend
                    orientation="horizontal"
                    items={legendData.map(item => item.title)}
                    colors={legendData.map(item => item.color)}
                    className="w-full horizontal-bar-legend"
                />
            </div>
        );
    }
}
