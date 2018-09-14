import React, { Component } from 'react';
import PropTypes from 'prop-types';
import dateFns from 'date-fns';
import dateTimeFormat from 'constants/dateTimeFormat';

import KeyValuePairs from 'Components/KeyValuePairs';
import CollapsibleCard from 'Components/CollapsibleCard';

const signalMap = {
    containerId: { label: 'Container' }
};

const processSignalsMap = {
    execFilePath: { label: 'Binary Path' },
    name: { label: 'Command Name' },
    args: { label: 'Arguments' },
    uid: { label: 'User ID' },
    gid: { label: 'Group ID' }
};

class ProcessDetails extends Component {
    static propTypes = {
        deployment: PropTypes.shape({ id: PropTypes.string.isRequired }).isRequired
    };

    renderProcess = process => {
        const processSignal = process.signal;
        let title = processSignal.execFilePath;
        const titleClassName =
            'p-3 border-b border-base-300 text-primary-600 tracking-wide cursor-pointer flex justify-between';
        if (process.signal.time) {
            title += ` | ${dateFns.format(process.signal.time, dateTimeFormat)}`;
        }
        return (
            <div className="px-3 py-4">
                <CollapsibleCard title={title} open={false} titleClassName={titleClassName}>
                    <div className="h-full p-3">
                        <KeyValuePairs data={process.signal} keyValueMap={signalMap} />
                        <KeyValuePairs data={process.signal} keyValueMap={processSignalsMap} />
                    </div>
                </CollapsibleCard>
            </div>
        );
    };

    renderProcesses = () => {
        const { deployment } = this.props;
        let processes = [];
        if (deployment.processes && deployment.processes.length !== 0) {
            processes = deployment.processes.map((process, index) => (
                <div key={index}>{this.renderProcess(process)}</div>
            ));
        } else {
            return (
                <div className="px-3 py-4">
                    <CollapsibleCard title="No Processes Found" />
                </div>
            );
        }
        return (
            <div className="px-3 py-4">
                <div className="h-full p-3">{processes}</div>
            </div>
        );
    };

    render() {
        return <div className="w-full">{this.renderProcesses()}</div>;
    }
}

export default ProcessDetails;
