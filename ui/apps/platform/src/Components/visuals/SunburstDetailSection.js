import React, { Component } from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import Truncate from 'react-truncate';
import { Link } from 'react-router-dom';

class SunburstDetailSection extends Component {
    static propTypes = {
        rootData: PropTypes.arrayOf(
            PropTypes.shape({
                text: PropTypes.string.isRequired,
                link: PropTypes.string,
                className: PropTypes.string,
            })
        ).isRequired,
        selectedDatum: PropTypes.shape({
            parent: PropTypes.shape({
                data: PropTypes.shape({
                    name: PropTypes.string,
                }),
            }),
            name: PropTypes.string,
        }),
        clicked: PropTypes.bool.isRequired,
        units: PropTypes.string,
    };

    static defaultProps = {
        selectedDatum: null,
        units: 'percentage',
    };

    getParentData = () => {
        const { selectedDatum } = this.props;
        if (selectedDatum) {
            const { parent } = selectedDatum;
            if (parent && parent.data && parent.data.name !== 'root') {
                return parent.data;
            }
        }
        return null;
    };

    getContent = () => {
        const { rootData, selectedDatum, units } = this.props;
        const parentDatum = this.getParentData();

        let bullets = [];

        if (selectedDatum) {
            if (parentDatum) {
                bullets.push({ text: parentDatum.name, ...parentDatum });
            }
            bullets.push({
                text: selectedDatum.name,
                ...selectedDatum,
            });
        } else {
            bullets = rootData;
        }
        return (
            <div className="py-2 px-3 lc:border-none lc:mb-0 lc:pb-0">
                {bullets.map(
                    ({
                        text,
                        link,
                        className,
                        color: graphColor,
                        textColor,
                        labelValue,
                        labelColor,
                    }) => {
                        const color = textColor || graphColor;
                        return (
                            <div
                                key={text}
                                className="widget-detail-bullet border-b border-base-300 pb-3 mb-1"
                            >
                                {link && (
                                    <Link
                                        title={text}
                                        className={`underline leading-normal flex w-full word-break ${
                                            className ?? ''
                                        }`}
                                        style={color ? { color } : null}
                                        to={link}
                                    >
                                        <Truncate lines={6} ellipsis={<>...</>}>
                                            {text}
                                        </Truncate>
                                    </Link>
                                )}
                                <span
                                    className="flex w-full word-break leading-tight"
                                    style={color ? { color } : null}
                                >
                                    <Truncate lines={4} ellipsis={<>...</>}>
                                        {!link && text}
                                    </Truncate>
                                </span>
                                {selectedDatum && units !== 'percentage' && (
                                    <span style={{ color: labelColor }}>{labelValue}</span>
                                )}
                            </div>
                        );
                    }
                )}
            </div>
        );
    };

    getLockHint = () => {
        const { clicked } = this.props;
        return (
            <div className="border-t border-base-300 border-dashed flex justify-end px-2 h-7 text-sm">
                <div className="flex items-center">
                    <Icon.Info size="16" className="pr-1" />
                    {`click to ${clicked ? 'un' : ''}lock selection`}
                </div>
            </div>
        );
    };

    render() {
        return (
            <div className="border-base-300 border-l flex flex-1 flex-col justify-between w-3/5 text-sm">
                {this.getContent()}
                {this.getLockHint()}
            </div>
        );
    }
}

export default SunburstDetailSection;
