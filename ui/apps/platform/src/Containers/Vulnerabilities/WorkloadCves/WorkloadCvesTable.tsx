import React from 'react';
import ImagesTableContainer from './Tables/ImagesTableContainer';

type WorkloadCvesTableProps = {
    entity: 'Image' | 'CVE' | 'Deployment';
};

function WorkloadCvesTable({ entity }: WorkloadCvesTableProps) {
    return (
        <>
            {entity === 'Image' && <ImagesTableContainer />}
            {entity === 'CVE' && <ImagesTableContainer />}
            {entity === 'Deployment' && <ImagesTableContainer />}
        </>
    );
}

export default WorkloadCvesTable;
