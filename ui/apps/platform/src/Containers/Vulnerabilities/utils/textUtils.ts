import { ensureExhaustive } from 'utils/type.utils';
import { Distro } from './sortUtils';

export function getDistroLinkText({ distro }: { distro: Distro }): string {
    switch (distro) {
        case 'rhel':
        case 'centos':
            return 'View in Red Hat CVE database';
        case 'ubuntu':
            return 'View in Ubuntu CVE database';
        case 'debian':
            return 'View in Debian CVE database';
        case 'alpine':
            return 'View in Alpine Linux CVE database';
        case 'amzn':
            return 'View in Amazon Linux CVE database';
        case 'other':
            return 'View additional information';
        default:
            return ensureExhaustive(distro);
    }
}
