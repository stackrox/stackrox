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
        fillOpacity: highlightedNames && !highlightedNames.includes(data.name) ? 0.3 : 1
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
        data: PropTypes.arrayOf(
            PropTypes.shape({
                name: PropTypes.string.isRequired,
                color: PropTypes.string.isRequired,
                link: PropTypes.string,
                value: PropTypes.number.isRequired,
                children: PropTypes.arrayOf(
                    PropTypes.shape({
                        name: PropTypes.string.isRequired,
                        color: PropTypes.string.isRequired,
                        link: PropTypes.string,
                        value: PropTypes.number.isRequired
                    })
                )
            })
        ).isRequired,
        rootData: PropTypes.arrayOf(
            PropTypes.shape({
                text: PropTypes.string.isRequired,
                link: PropTypes.string
            })
        ).isRequired,
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
        const data = merge({}, props.data);
        const enrichedData = this.enrichData(data);
        this.state = {
            data: enrichedData,
            clicked: false,
            selectedDatum: null
        };
    }

    getCenterLabel = () => {
        const { data } = this.props;
        const val = data.reduce(
            (acc, curr) => ({ total: acc.total + 100, passing: acc.passing + curr.value }),
            {
                total: 0,
                passing: 0
            }
        );
        const label = Math.round((val.passing / val.total) * 100);
        return <LabelSeries data={[{ x: 0, y: 10, label: `${label}%`, style: LABEL_STYLE }]} />;
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

    enrichData = data => {
        const enrichedData = {
            title: 'Root Title',
            name: 'root',
            color: 'var(--base-100)',
            children: data.map(({ children, ...rest }) => {
                const result = {
                    ...rest,
                    radius: 20,
                    radius0: 60,
                    stroke: 2,
                    style: {
                        stroke: 'var(--base-100)'
                    },
                    title: 'Inner Title',
                    children: children.map(({ ...props }) => {
                        const childResult = {
                            ...props,
                            radius: 60,
                            radius0: 100,
                            size: 1,
                            style: {
                                stroke: 'var(--base-100)',
                                fillOpacity: 1
                            },
                            title: 'Outer Title'
                        };
                        return childResult;
                    })
                };
                return result;
            })
        };
        return enrichedData;
    };

    render() {
        const { legendData, rootData } = this.props;
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
                <SunburstDetailSection
                    selectedDatum={selectedDatum}
                    rootData={rootData}
                    clicked={clicked}
                />
            </>
        );
    }
}
