import PreviewLabelBase from './PreviewLabelBase';

export function DeveloperPreviewLabel() {
    return (
        <PreviewLabelBase
            ariaLabel="Developer preview info"
            title="Developer preview"
            color="purple"
            body="Developer preview features are not intended to be used in production environments. The clusters deployed with the developer preview features are considered development clusters and are not supported through the Red Hat Customer Portal case management system."
        />
    );
}

export default DeveloperPreviewLabel;
