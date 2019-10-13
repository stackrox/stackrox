import VulnMgmtList from 'Containers/VulnMgmt/List/VulnMgmtList';
import VulnMgmtEntity from 'Containers/VulnMgmt/Entity/VulnMgmtEntity';
import useCaseTypes from 'constants/useCaseTypes';

export const ListComponentMap = {
    [useCaseTypes.VULN_MANAGEMENT]: VulnMgmtList
};

export const EntityComponentMap = {
    [useCaseTypes.VULN_MANAGEMENT]: VulnMgmtEntity
};
