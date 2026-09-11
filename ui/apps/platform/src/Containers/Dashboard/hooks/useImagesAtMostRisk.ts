import { useCallback } from 'react';

import { getImage, listImages } from 'services/imageService';
import type { ImageDetails } from 'services/imageService';
import type { ListImage } from 'types/image.proto';
import useRestQuery from 'hooks/useRestQuery';

import type { ImageData } from '../Widgets/ImagesAtMostRiskTable';

function countImageVulnerabilityCounter(
    scan: ImageDetails['scan']
): ImageData['images'][number]['imageVulnerabilityCounter'] {
    const byCve = new Map<string, { severity: string; fixable: boolean }>();

    scan?.components?.forEach((component) => {
        component.vulns?.forEach((vuln) => {
            const existing = byCve.get(vuln.cve);
            const fixable = Boolean(vuln.fixedBy);
            if (!existing) {
                byCve.set(vuln.cve, { severity: vuln.severity, fixable });
            } else if (fixable) {
                existing.fixable = true;
            }
        });
    });

    const counts = {
        important: { total: 0, fixable: 0 },
        critical: { total: 0, fixable: 0 },
    };

    byCve.forEach(({ severity, fixable }) => {
        if (severity === 'CRITICAL_VULNERABILITY_SEVERITY') {
            counts.critical.total += 1;
            if (fixable) {
                counts.critical.fixable += 1;
            }
        } else if (severity === 'IMPORTANT_VULNERABILITY_SEVERITY') {
            counts.important.total += 1;
            if (fixable) {
                counts.important.fixable += 1;
            }
        }
    });

    return counts;
}

function remoteFromFullName(fullName: string): string {
    const withoutTag = fullName.replace(/:([^:/]+)$/, '');
    const slashIndex = withoutTag.indexOf('/');
    return slashIndex === -1 ? withoutTag : withoutTag.slice(slashIndex + 1);
}

function toImageData(listImage: ListImage, details: ImageDetails): ImageData['images'][number] {
    return {
        id: details.id || listImage.id,
        name: {
            remote: details.name?.remote ?? remoteFromFullName(listImage.name),
            fullName: details.name?.fullName ?? listImage.name,
        },
        priority: Number(details.priority ?? listImage.priority),
        imageVulnerabilityCounter: countImageVulnerabilityCounter(details.scan),
    };
}

async function fetchImagesAtMostRisk(query: string): Promise<ImageData> {
    const list = await listImages({
        query,
        sortOption: { field: 'Image Risk Priority', reversed: false },
        page: 1,
        perPage: 6,
    });
    const details = await Promise.all(list.map((image) => getImage(image.id)));
    return {
        images: details.map((image, index) => toImageData(list[index], image)),
    };
}

export default function useImagesAtMostRisk(query: string) {
    const restQuery = useCallback(() => fetchImagesAtMostRisk(query), [query]);
    return useRestQuery(restQuery);
}
