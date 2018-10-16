import React, { Component } from 'react';
import PropTypes from 'prop-types';

import ProcessesCollapsibleCard from 'Containers/Violations/ProcessesCollapsibleCard';

import * as Icon from 'react-feather';
import { getTime, format } from 'date-fns';
import dateTimeFormat from 'constants/dateTimeFormat';

class ViolationsDetails extends Component {
    static propTypes = {
        violations: PropTypes.arrayOf(
            PropTypes.shape({
                message: PropTypes.string.isRequired
            })
        ),
        firstOccurred: PropTypes.string.isRequired
    };

    static defaultProps = {
        violations: []
    };

    getDeploytimeMessages = () => {
        const { violations } = this.props;
        return violations.filter(violation => !violation.processes.length).map(({ message }) => (
            <div
                key={message}
                className="mb-4 p-3 pb-2 shadow border border-base-200 text-base-600 flex justify-between leading-normal bg-base-100"
            >
                {message}
            </div>
        ));
    };

    getRuntimeMessages = () => {
        const { violations, firstOccurred } = this.props;
        return violations
            .filter(violation => violation.processes.length)
            .map(({ message, processes }) => {
                const firstOccurrenceTimestamp = getTime(firstOccurred);
                const lastOccurrenceTimestamp = Math.max(
                    ...processes.map(process => getTime(process.signal.time))
                );
                const processesList = processes.map((process, index) => {
                    const { time, args, execFilePath, containerId } = process.signal;
                    const processTime = new Date(time);
                    const timeFormat = format(processTime, dateTimeFormat);
                    return (
                        <div className="border-t border-base-300" key={process.id}>
                            <div className="flex text-base-600">
                                <span className="flex items-center bg-caution-300 font-700 px-2">
                                    {index + 1}
                                </span>
                                <span className="py-2 px-2 bg-caution-200">{execFilePath}</span>
                            </div>
                            <div className="flex flex-1 text-base-600 px-4 py-2 justify-between">
                                <div>
                                    <span className="font-700">Container ID:</span> {containerId}
                                </div>
                                <div>
                                    <span className="font-700">Time:</span> {timeFormat}
                                </div>
                            </div>
                            <div className="flex flex-1 text-base-600 px-4 py-2">
                                <div>
                                    <span className="font-700">Arguments:</span> {args}
                                </div>
                            </div>
                        </div>
                    );
                });
                return (
                    <div className="mb-4" key={message}>
                        <ProcessesCollapsibleCard title={message}>
                            <div>
                                <div className="flex flex-1 bg-primary-100">
                                    <div className="w-1/2 p-4 border-r border-base-300 leading-normal">
                                        <div className="flex justify-center font-700 italic">
                                            First Occurence:
                                        </div>
                                        <div className="flex justify-center font-600">
                                            {format(firstOccurrenceTimestamp, dateTimeFormat)}
                                        </div>
                                    </div>
                                    <div className="w-1/2 p-4 leading-normal">
                                        <div className="flex justify-center font-700 italic">
                                            Last Occurence:
                                        </div>
                                        <div className="flex justify-center font-600">
                                            {format(lastOccurrenceTimestamp, dateTimeFormat)}
                                        </div>
                                    </div>
                                </div>
                                <div>{processesList}</div>
                            </div>
                        </ProcessesCollapsibleCard>
                    </div>
                );
            });
    };

    render() {
        const deploytimeMessages = this.getDeploytimeMessages();
        const runtimeMessages = this.getRuntimeMessages();
        let separator = null;
        if (deploytimeMessages.length && runtimeMessages.length) {
            separator = (
                <div className="flex justify-center items-center mt-4">
                    <Icon.Plus className="h-8 w-8 text-base-400" />
                </div>
            );
        }
        return (
            <div className="w-full px-3 pb-5 mt-5">
                {runtimeMessages}
                {separator}
                {deploytimeMessages}
            </div>
        );
    }
}

export default ViolationsDetails;
