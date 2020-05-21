import React from 'react';
import PropTypes from 'prop-types';

import { knownBackendFlags } from 'utils/featureFlags';
import FeatureEnabled from 'Containers/FeatureEnabled';
import ProcessComments from 'Containers/AnalystNotes/ProcessComments';
import ProcessTags from 'Containers/AnalystNotes/ProcessTags';
import ProcessSignal from './Signal';
import BinaryCollapsible from './BinaryCollapsible';

function Binaries({ processes }) {
    return processes.map(({ args, signals }) => {
        /* For process groups we're going to be taking any one process within the grouping
         * and use the "deploymentId", "containerName", "execFilePath", and "args" as ids
         * for comments/tags. Unfortunately, because of SAC restrictions, backend can’t
         * just take a group id. When comments/tags are added for any process within the
         * group, all the processes will update as well.
         * */
        const { deploymentId, containerName } = signals[0];
        const { execFilePath } = signals[0].signal;
        return (
            <BinaryCollapsible commandLineArgs={args} key={args}>
                <FeatureEnabled featureFlag={knownBackendFlags.ROX_ANALYST_NOTES_UI}>
                    <div className="pt-4 px-4">
                        <ProcessTags
                            deploymentID={deploymentId}
                            containerName={containerName}
                            execFilePath={execFilePath}
                            args={args}
                        />
                    </div>
                    <div className="py-4 px-4">
                        <ProcessComments
                            deploymentID={deploymentId}
                            containerName={containerName}
                            execFilePath={execFilePath}
                            args={args}
                        />
                    </div>
                </FeatureEnabled>
                <ProcessSignal signals={signals} />
            </BinaryCollapsible>
        );
    });
}

Binaries.propTypes = {
    processes: PropTypes.arrayOf(
        PropTypes.shape({
            args: PropTypes.string,
            signals: PropTypes.arrayOf(PropTypes.object),
        })
    ).isRequired,
};

export default Binaries;
