// We can move this out to a higher level if more components need it
import { useEffect, useState } from 'react';
import { Role, fetchRolesAsArray } from 'services/RolesService';
import {fetchAllowedRoles} from "../../../../../services/APITokensService";

export type UseRolesResult = {
    roles: string[];
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
        fetchAllowedRoles()
            .then((roles) => {
                setResult({ roles: roles.response.roles, error: null, isLoading: false });
            })
            .catch((error) => {
                setResult({ roles: [], error, isLoading: false });
            });
    }, []);

    return result;
};

export default useRoles;
