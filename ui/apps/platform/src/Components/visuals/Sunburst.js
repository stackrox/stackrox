import React from 'react';
import { Sunburst, DiscreteColorLegend, LabelSeries } from 'react-vis';
import PropTypes from 'prop-types';
import merge from 'deepmerge';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';

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
        data.children.map((child) => highlightPathData(child, highlightedNames));
    }
    // eslint-disable-next-line no-param-reassign
    data.style = {
        ...data.style,
        fillOpacity: highlightedNames && !highlightedNames.includes(data.name) ? 0.3 : 1,
    };
    return data;
}

const LABEL_STYLE = {
    fontSize: '12px',
    textAnchor: 'middle',
    fill: 'var(--primary-800)',
};

class BasicSunburst extends React.Component {
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
                        value: PropTypes.number.isRequired,
                    })
                ),
            })
        ).isRequired,
        rootData: PropTypes.arrayOf(
            PropTypes.shape({
                text: PropTypes.string.isRequired,
                link: PropTypes.string,
            })
        ).isRequired,
        totalValue: PropTypes.number.isRequired,
        legendData: PropTypes.arrayOf(PropTypes.object),
        sunburstProps: PropTypes.shape({
            style: PropTypes.string,
        }),
        onValueMouseOver: PropTypes.func,
        onValueMouseOut: PropTypes.func,
        onValueSelect: PropTypes.func,
        onValueDeselect: PropTypes.func,
        staticDetails: PropTypes.bool,
        history: ReactRouterPropTypes.history.isRequired,
        units: PropTypes.string,
        small: PropTypes.bool,
    };

    static defaultProps = {
        sunburstProps: {},
        legendData: null,
        onValueMouseOver: null,
        onValueMouseOut: null,
        onValueSelect: null,
        onValueDeselect: null,
        staticDetails: false,
        units: 'percentage',
        small: false,
    };

    constructor(props) {
        super(props);
        const data = merge({}, props.data);
        const enrichedData = this.enrichData(data);
        this.state = {
            data: enrichedData,
            clicked: false,
            selectedDatum: null,
        };
    }

    getCenterLabel = () => {
        const label = `${this.props.totalValue}${this.props.units === 'percentage' ? '%' : ''}`;
        return <LabelSeries data={[{ x: 1, y: 9, label, style: LABEL_STYLE }]} />;
    };

    onValueMouseOverHandler = (datum) => {
        const { data, clicked } = this.state;
        const { onValueMouseOver } = this.props;
        if (clicked) {
            return;
        }
        const path = getKeyPath(datum);
        this.setState({
            data: highlightPathData(data, path),
            selectedDatum: datum,
        });
        if (onValueMouseOver) {
            onValueMouseOver(path);
        }
    };

    onValueMouseOutHandler = () => {
        const { data, clicked } = this.state;
        const { onValueMouseOut } = this.props;
        if (clicked) {
            return;
        }
        this.setState({
            selectedDatum: null,
            data: highlightPathData(data, false),
        });
        if (onValueMouseOut) {
            onValueMouseOut();
        }
    };

    onValueClickHandler = (datum) => {
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
            // TODO: factor out into dimension mapping
            width: this.props.small ? 200 : 265,
            height: this.props.small ? 200 : 271,
            className: 'cursor-pointer pointer-events-none my-auto',
            onValueMouseOver: this.onValueMouseOverHandler,
            onValueMouseOut: this.onValueMouseOutHandler,
            onValueClick: this.onValueClickHandler,
        };
        return merge(defaultSunburstProps, this.props.sunburstProps);
    };

    enrichData = (data) => {
        const enrichedData = {
            title: 'Root Title',
            name: 'root',
            color: 'var(--base-100)',
            children: data.map(({ children, ...rest }) => {
                const result = {
                    ...rest,
                    radius: this.props.small ? 15 : 20,
                    radius0: this.props.small ? 50 : 60,
                    stroke: 2,
                    style: {
                        stroke: 'var(--base-100)',
                    },
                    title: 'Inner Title',
                    children: children.map(({ ...props }) => {
                        const childResult = {
                            ...props,
                            radius: this.props.small ? 50 : 60,
                            radius0: this.props.small ? 90 : 120,
                            size: 1,
                            style: {
                                stroke: 'var(--base-100)',
                                fillOpacity: 1,
                            },
                            title: 'Outer Title',
                        };
                        return childResult;
                    }),
                };
                return result;
            }),
        };
        return enrichedData;
    };

    render() {
        const { legendData, rootData, staticDetails } = this.props;
        const { clicked, data, selectedDatum } = this.state;

        const sunburstProps = this.getSunburstProps();
        const sunburstStyle = {
            stroke: '#ddd',
            strokeOpacity: 0.3,
            strokeWidth: '0.5',
            ...this.props.sunburstProps.style,
        };
        sunburstProps.style = sunburstStyle;

        return (
            <>
                <div className="flex flex-col justify-between">
                    <Sunburst data={data} {...sunburstProps} hideRootNode>
                        {this.getCenterLabel()}
                    </Sunburst>
                    {legendData && (
                        <DiscreteColorLegend
                            orientation="horizontal"
                            items={legendData}
                            className="w-full horizontal-bar-legend border-t border-base-300 h-7 flex justify-between items-center"
                        />
                    )}
                </div>
                <SunburstDetailSection
                    selectedDatum={!staticDetails ? selectedDatum : null}
                    rootData={rootData}
                    clicked={clicked}
                    units="value"
                />
            </>
        );
    }
}

export default withRouter(BasicSunburst);
