import { useEffect, useState } from 'react';

import { MitreAttackVector, fetchMitreAttackVectors } from 'services/MitreService';

export type UseFetchMitreAttackVectorsResult = {
    mitreAttackVectors: MitreAttackVector[];
    error: string | null;
    isLoading: boolean;
};

const defaultResult = {
    mitreAttackVectors: [],
    error: null,
    isLoading: false,
};

const useFetchMitreAttackVectors = (): UseFetchMitreAttackVectorsResult => {
    const [result, setResult] = useState<UseFetchMitreAttackVectorsResult>(defaultResult);

    useEffect(() => {
        setResult((prevResult) => ({ ...prevResult, isLoading: true }));
        fetchMitreAttackVectors()
            .then((mitreAttackVectors) => {
                setResult({ mitreAttackVectors, error: null, isLoading: false });
            })
            .catch((error) => {
                setResult({ mitreAttackVectors: [], error, isLoading: false });
            });
    }, []);

    return result;
};

export default useFetchMitreAttackVectors;
