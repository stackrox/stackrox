import React from 'react';
import type { ReactElement } from 'react';

import CollectorStatusLegacy from './Collector/CollectorStatusLegacy';
import AdmissionControlStatusLegacy from './AdmissionControl/AdmissionControlStatusLegacy';
import SensorStatusLegacy from './SensorStatusLegacy';
import type { ClusterHealthStatus } from '../clusterTypes';
import ScannerStatusLegacy from './Scanner/ScannerStatusLegacy';

type ClusterStatusPillProps = {
    healthStatus: ClusterHealthStatus;
};

function ClusterStatusPill({ healthStatus }: ClusterStatusPillProps): ReactElement {
    const scannerHealthStatus = healthStatus?.scannerHealthStatus || 'UNINITIALIZED';

    return (
        <div className="border inline rounded-full decoration-clone leading-looser text-sm py-1 word-break">
            <div className="inline border-r pl-2 pr-3 w-full whitespace-nowrap">
                <CollectorStatusLegacy healthStatus={healthStatus} isList />
            </div>
            <div className="inline border-r pl-2 pr-3 w-full whitespace-nowrap">
                <SensorStatusLegacy healthStatus={healthStatus} isList />
            </div>
            <div className="inline pl-2 pr-3 w-full whitespace-nowrap">
                <AdmissionControlStatusLegacy healthStatus={healthStatus} isList />
            </div>
            {scannerHealthStatus !== 'UNINITIALIZED' && (
                <div className="inline border-l pl-2 pr-3 w-full whitespace-nowrap">
                    <ScannerStatusLegacy healthStatus={healthStatus} isList />
                </div>
            )}
        </div>
    );
}

export default ClusterStatusPill;
