// We can move this out to a higher level if more components need it
import { useEffect, useState } from 'react';
import { Role, fetchRolesAsArray } from 'services/RolesService';

export type UseRolesResult = {
    roles: Role[];
    error: string | null;
    isLoading: boolean;
};

const defaultResult = {
    roles: [],
    error: null,
    isLoading: false,
};

const useRoles = (): UseRolesResult => {
    const [result, setResult] = useState<UseRolesResult>(defaultResult);

    useEffect(() => {
        setResult((prevResult) => ({ ...prevResult, isLoading: true }));
        fetchRolesAsArray()
            .then((roles) => {
                setResult({ roles, error: null, isLoading: false });
            })
            .catch((error) => {
                setResult({ roles: [], error, isLoading: false });
            });
    }, []);

    return result;
};

export default useRoles;
