// We can move this out to a higher level if more components need it
import { useEffect, useState } from 'react';
import { fetchAllowedRoles } from 'services/APITokensService';

export type UseRolesResult = {
    roleNames: string[];
    error: string | null;
    isLoading: boolean;
};

const defaultResult = {
    roleNames: [],
    error: null,
    isLoading: false,
};

const useAllowedRoles = (): UseRolesResult => {
    const [result, setResult] = useState<UseRolesResult>(defaultResult);

    useEffect(() => {
        setResult((prevResult) => ({ ...prevResult, isLoading: true }));
        fetchAllowedRoles()
            .then((roleNames) => {
                setResult({ roleNames, error: null, isLoading: false });
            })
            .catch((error) => {
                setResult({ roleNames: [], error, isLoading: false });
            });
    }, []);

    return result;
};

export default useAllowedRoles;
