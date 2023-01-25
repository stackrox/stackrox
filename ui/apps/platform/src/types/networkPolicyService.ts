// Some types in this file were modified for readability.
// Check out proto/api/v1/network_policy_service.proto for the proper names
import { NetworkBaselineConnectionProperties } from './networkBaseline.proto';
import { NetworkEntityInfo } from './networkFlow.proto';

export type ReconciledDiffFlows = {
    entity: NetworkEntityInfo;
    added: NetworkBaselineConnectionProperties[];
    removed: NetworkBaselineConnectionProperties[];
    unchanged: NetworkBaselineConnectionProperties[];
};

export type GroupedDiffFlows = {
    entity: NetworkEntityInfo;
    properties: NetworkBaselineConnectionProperties[];
};

export type DiffFlowsResponse = {
    added: GroupedDiffFlows[];
    removed: GroupedDiffFlows[];
    reconciled: ReconciledDiffFlows[];
};
