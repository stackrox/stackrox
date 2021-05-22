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
        <div className="border inline rounded-full decoration-clone leading-looser text-sm py-1">
            <div className="inline border-r pl-2 pr-3 w-full whitespace-nowrap">
                <CollectorStatus healthStatus={healthStatus} isList />
            </div>
            <div className="inline border-r pl-2 pr-3 w-full whitespace-nowrap">
                <SensorStatus healthStatus={healthStatus} isList />
            </div>
            <div className="inline pl-2 pr-3 w-full whitespace-nowrap">
                <AdmissionControlStatus healthStatus={healthStatus} isList />
            </div>
        </div>
    );
}

export default ClusterStatusPill;
