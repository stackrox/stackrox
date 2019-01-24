import React, { Component } from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';

import HorizontalBarChart from 'Components/visuals/HorizontalBar';
import { horizontalBarDatum } from 'mockData/graphDataMock';

function formatAsPercent(x) {
    return `${x}%`;
}

class SunburstDetailSection extends Component {
    static propTypes = {
        selectedDatum: PropTypes.shape({}),
        clicked: PropTypes.bool.isRequired
    };

    static defaultProps = {
        selectedDatum: null
    };

    getParentText = () => {
        const { selectedDatum } = this.props;
        if (selectedDatum) {
            const { parent } = selectedDatum;
            if (parent && parent.data && parent.data.name !== 'root') {
                return parent.data.name;
            }
        }
        return null;
    };

    getContent = () => {
        const { selectedDatum } = this.props;
        const parentText = this.getParentText();

        let bullets = [];

        if (selectedDatum) {
            if (parentText) bullets.push({ text: parentText });
            bullets.push({ text: selectedDatum.name });
        } else
            bullets = [
                {
                    text: '12 categories'
                },
                { text: '43 controls', link: 'https://google.com' },
                { text: '29 passed controls', link: 'https://google.com/' },
                { text: '14 failed controls' }
            ];

        return (
            <div className="pt-3 pl-3">
                {bullets.map(({ text, link }, idx) => (
                    <div
                        key={text}
                        className={`widget-detail-bullet ${
                            parentText && idx === 0 ? 'text-base-500' : ''
                        }`}
                    >
                        {link && (
                            <a className="underline text-base-600" href={link}>
                                {text}
                            </a>
                        )}
                        {!link && text}
                        {selectedDatum && (
                            <HorizontalBarChart
                                data={horizontalBarDatum}
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
            <div className="border-t border-base-300 border-dashed flex justify-center h-7 text-base-500 text-sm">
                <div className="flex self-center">
                    <Icon.Info className="h-3 w-3 pr-1" />
                    {`click to ${clicked ? 'un' : ''}lock selection`}
                </div>
            </div>
        );
    };

    render() {
        return (
            <div className="border-base-300 border-l flex flex-col justify-between w-1/3">
                {this.getContent()}
                {this.getLockHint()}
            </div>
        );
    }
}

export default SunburstDetailSection;
