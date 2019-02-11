import React, { Component } from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import Truncate from 'react-truncate';
import { Link } from 'react-router-dom';
import HorizontalBarChart from 'Components/visuals/HorizontalBar';

function formatAsPercent(x) {
    return `${x}%`;
}

class SunburstDetailSection extends Component {
    static propTypes = {
        rootData: PropTypes.arrayOf(
            PropTypes.shape({
                text: PropTypes.string.isRequired,
                link: PropTypes.string
            })
        ).isRequired,
        selectedDatum: PropTypes.shape({}),
        clicked: PropTypes.bool.isRequired
    };

    static defaultProps = {
        selectedDatum: null
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
        const { rootData, selectedDatum } = this.props;
        const parentDatum = this.getParentData();

        let bullets = [];

        if (selectedDatum) {
            if (parentDatum) bullets.push({ text: parentDatum.name, ...parentDatum });
            bullets.push({
                text: selectedDatum.name,
                ...selectedDatum
            });
        } else {
            bullets = rootData;
        }

        return (
            <div className="py-2 px-3 fc:border-b fc:border-base-300 fc:pb-3 fc:mb-1">
                {bullets.map(({ text, link, value }, idx) => (
                    <div
                        key={text}
                        className={`widget-detail-bullet font-600 ${
                            parentDatum && parentDatum.name && idx === 0 ? 'text-base-500' : ''
                        }`}
                    >
                        {link && (
                            <Link
                                className="underline text-base-600 hover:text-primary-700 leading-normal flex w-full word-break"
                                to={link}
                            >
                                <Truncate lines={6} ellipsis={<>...</>}>
                                    {text}
                                </Truncate>
                            </Link>
                        )}
                        <span className="flex w-full word-break leading-tight">
                            <Truncate lines={4} ellipsis={<>...</>}>
                                {!link && text}
                            </Truncate>
                        </span>
                        {selectedDatum && (
                            <HorizontalBarChart
                                data={[{ y: '', x: value }]}
                                valueFormat={formatAsPercent}
                                minimal
                            />
                        )}
                    </div>
                ))}
            </div>
        );
    };

    getLockHint = () => {
        const { clicked } = this.props;
        return (
            <div className="border-t border-base-300 border-dashed flex justify-end px-2 h-7 text-base-500 text-sm">
                <div className="flex items-center font-condensed opacity-75">
                    <Icon.Info size="16" className="pr-1" />
                    {`click to ${clicked ? 'un' : ''}lock selection`}
                </div>
            </div>
        );
    };

    render() {
        return (
            <div className="border-base-300 border-l flex flex-col justify-between w-3/5">
                {this.getContent()}
                {this.getLockHint()}
            </div>
        );
    }
}

export default SunburstDetailSection;
