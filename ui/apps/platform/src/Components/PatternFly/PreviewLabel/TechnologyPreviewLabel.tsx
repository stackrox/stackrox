import PreviewLabelBase from './PreviewLabelBase';

// Render TechPreviewLabel when width is limited: in left navigation or integration tile.
// Render TechnologyPreviewLabel when when width is not limited: in heading.
export function TechnologyPreviewLabel() {
    return (
        <PreviewLabelBase
            ariaLabel="Technology preview info"
            title="Technology preview"
            body="Technology Preview features provide early access to upcoming product innovations, enabling you to test functionality and provide feedback during the development process."
        />
    );
}

export default TechnologyPreviewLabel;
