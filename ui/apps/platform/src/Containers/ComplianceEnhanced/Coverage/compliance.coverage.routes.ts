import { complianceEnhancedCoveragePath } from 'routePaths';

export const coverageProfileChecksPath = `${complianceEnhancedCoveragePath}/profiles/:profileName/checks`;
export const coverageProfileClustersPath = `${complianceEnhancedCoveragePath}/profiles/:profileName/clusters`;
export const coverageCheckDetailsPath = `${coverageProfileChecksPath}/:checkName`;
export const coverageClusterDetailsPath = `${coverageProfileClustersPath}/:clusterName`;
