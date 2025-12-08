import { useContext } from 'react';
import { WorkloadCveViewContext } from '../WorkloadCveViewContext';
import type { WorkloadCveView } from '../WorkloadCveViewContext';

export default function useWorkloadCveViewContext(): WorkloadCveView {
    const value = useContext(WorkloadCveViewContext);

    if (!value) {
        throw new Error('A value must be provided to the WorkloadCveViewContext via a provider!');
    }

    return value;
}
