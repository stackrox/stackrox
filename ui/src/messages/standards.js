import { standardTypes } from 'constants/entityTypes';

export const standardLabels = {
    [standardTypes.PCI_DSS_3_2]: 'PCI DSS 3.2',
    [standardTypes.NIST_800_190]: 'NIST 800-190',
    [standardTypes.HIPAA_164]: 'HIPAA 164',
    [standardTypes.CIS_Kubernetes_v1_2_0]: 'CIS Kubernetes v1.2.0',
    [standardTypes.CIS_Docker_v1_1_0]: 'CIS Docker v1.1.0'
};

export const standardShortLabels = {
    ...standardLabels,
    [standardTypes.CIS_Kubernetes_v1_2_0]: 'CIS K8s v1.2.0'
};

export default standardLabels;
