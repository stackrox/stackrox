import React from 'react';
import { useParams } from 'react-router-dom';

function WorkloadCvesSinglePage() {
    const { cveId } = useParams();
    return <>Workload CVE Single Page: {cveId}</>;
}

export default WorkloadCvesSinglePage;
