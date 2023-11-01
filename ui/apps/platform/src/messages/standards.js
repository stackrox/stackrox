import { standardTypes } from 'constants/entityTypes';

export const standardLabels = {
    [standardTypes.PCI_DSS_3_2]: 'PCI DSS 3.2.1',
    [standardTypes.NIST_800_190]: 'NIST SP 800-190',
    [standardTypes.NIST_SP_800_53_Rev_4]: 'NIST SP 800-53',
    [standardTypes.HIPAA_164]: 'HIPAA 164',
    [standardTypes.CIS_Kubernetes_v1_5]: 'CIS Kubernetes v1.5'
};

export const standardShortLabels = {
    ...standardLabels,
    [standardTypes.CIS_Kubernetes_v1_5]: 'CIS K8s v1.5',
};

export default standardLabels;
