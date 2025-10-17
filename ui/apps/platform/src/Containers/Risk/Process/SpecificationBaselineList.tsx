import React from 'react';
import type { ProcessBaseline } from 'types/processBaseline.proto';

import ProcessBaselineList from './ProcessBaselineList';

export type SpecificationBaselineListProps = {
    processBaselines: ProcessBaseline[];
    processEpoch: number;
    setProcessEpoch: (number) => void;
};

function SpecificationBaselineList({
    processBaselines,
    processEpoch,
    setProcessEpoch,
}: SpecificationBaselineListProps) {
    return (
        <div className="pl-3 pr-3">
            <ul className="border-b border-base-300 leading-normal hover:bg-primary-100">
                {processBaselines.map((data) => (
                    <ProcessBaselineList
                        process={data}
                        key={data.key.containerName}
                        processEpoch={processEpoch}
                        setProcessEpoch={setProcessEpoch}
                    />
                ))}
            </ul>
        </div>
    );
}

export default SpecificationBaselineList;
