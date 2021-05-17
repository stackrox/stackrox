import React, { ReactElement } from 'react';

import CollectorStatus from './Collector/CollectorStatus';
import AdmissionControlStatus from './AdmissionControl/AdmissionControlStatus';
import SensorStatus from './SensorStatus';
import { ClusterHealthStatus } from '../clusterTypes';

type ClusterStatusPillProps = {
    healthStatus: ClusterHealthStatus;
};

function ClusterStatusPill({ healthStatus }: ClusterStatusPillProps): ReactElement {
    return (
        <div className="border rounded-full flex mt-1">
            <div className="pl-1 pr-2">
                <CollectorStatus healthStatus={healthStatus} isList />
            </div>
            <div className="border-l pl-1 pr-2">
                <SensorStatus healthStatus={healthStatus} isList />
            </div>
            <div className="border-l pl-1 pr-2">
                <AdmissionControlStatus healthStatus={healthStatus} isList />
            </div>
        </div>
    );
}

export default ClusterStatusPill;
