import React from 'react';
import { useParams } from 'react-router-dom';

function WorkloadCvesImageSinglePage() {
    const { imageId } = useParams();
    return <>Workload CVE Image Single Page: {imageId}</>;
}

export default WorkloadCvesImageSinglePage;
