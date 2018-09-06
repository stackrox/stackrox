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
    name: { label: 'Name' },
    commandLine: { label: 'Command Line' },
    execFilePath: { label: 'Process path' }
};

class ProcessDetails extends Component {
    static propTypes = {
        deployment: PropTypes.shape({ id: PropTypes.string.isRequired }).isRequired
    };

    renderProcess = process => {
        const { processSignal } = process.signal;
        let title = processSignal.name;
        if (process.signal.time) {
            title += `|${dateFns.format(process.signal.time, dateTimeFormat)}`;
        }
        return (
            <div className="px-3 py-4">
                <CollapsibleCard title={title}>
                    <div className="h-full p-3">
                        <KeyValuePairs data={process.signal} keyValueMap={signalMap} />
                        <KeyValuePairs
                            data={process.signal.processSignal}
                            keyValueMap={processSignalsMap}
                        />
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
