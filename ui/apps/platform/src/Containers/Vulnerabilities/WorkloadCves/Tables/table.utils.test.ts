import { flattenDeploymentComponentVulns, flattenImageComponentVulns } from './table.utils';
import type {
    DeploymentComponentVulnerability,
    ImageComponentVulnerability,
    ImageMetadataContext,
} from './table.utils';

const image: ImageMetadataContext = {
    id: 'image-id',
    name: { registry: 'registry.example.com', remote: 'app', tag: 'latest' },
    metadata: null,
};

function makeComponents(cvss: number | null) {
    const imageComponent: ImageComponentVulnerability = {
        type: 'Image',
        name: 'php',
        version: '8.3.35',
        location: '/usr/bin/php',
        source: 'OS',
        layerIndex: null,
        imageVulnerabilities: [
            {
                severity: 'IMPORTANT_VULNERABILITY_SEVERITY',
                cvss,
                scoreVersion: 'V3',
                fixedByVersion: '',
                pendingExceptionCount: 0,
            },
        ],
    };
    const deploymentComponent: DeploymentComponentVulnerability = {
        ...imageComponent,
        type: 'Deployment',
        imageVulnerabilities: imageComponent.imageVulnerabilities.map((vulnerability) => ({
            ...vulnerability,
            discoveredAtImage: null,
            publishedOn: null,
        })),
    };

    return { imageComponent, deploymentComponent };
}

describe('component vulnerability rows', () => {
    const scores = { unavailable: null, zero: 0, scored: 7.234 };

    it.each(Object.entries(scores))('preserves %s instance CVSS in both table paths', (_, cvss) => {
        const { imageComponent, deploymentComponent } = makeComponents(cvss);

        const [imageRow] = flattenImageComponentVulns(image, [imageComponent]);
        const [deploymentRow] = flattenDeploymentComponentVulns(image, [deploymentComponent]);

        expect(imageRow.cvss).toBe(cvss);
        expect(deploymentRow.cvss).toBe(cvss);
    });

    it('preserves absence when a component has no matching vulnerability', () => {
        const { imageComponent, deploymentComponent } = makeComponents(null);
        imageComponent.imageVulnerabilities = [];
        deploymentComponent.imageVulnerabilities = [];

        const [imageRow] = flattenImageComponentVulns(image, [imageComponent]);
        const [deploymentRow] = flattenDeploymentComponentVulns(image, [deploymentComponent]);

        expect(imageRow.cvss).toBeNull();
        expect(deploymentRow.cvss).toBeNull();
    });
});
